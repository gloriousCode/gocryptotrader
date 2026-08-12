package log

import (
	"errors"
	"fmt"
	"io"
	"log"
	"slices"
	"time"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var (
	errWriterAlreadyLoaded     = errors.New("io.Writer already loaded")
	errJobsChannelIsFull       = errors.New("logger jobs channel is filled")
	errWriterIsNil             = errors.New("io writer is nil")
	message                Key = "message"
	timestamp              Key = "timestamp"
	severity               Key = "severity"
	subLoggerName          Key = "sublogger"
	botName                Key = "botname"
)

type writerBatch struct {
	writer io.Writer
	data   []byte
}

// loggerWorker handles all work staged to be written to configured io.Writer(s)
// This worker is generated in init() to handle full workload.
func loggerWorker() {
	buffer := make([]byte, 0, defaultBufferCapacity)
	batchData := make([]writerBatch, 0, adaptiveBatchMaxJobs)
	structuredOutbound := ExtraFields{}
	for j := range jobsChannel {
		if j.Passback != nil {
			j.Passback <- struct{}{}
			continue
		}
		backlog := len(jobsChannel)
		if backlog < adaptiveBatchTrigger {
			processSingleLogJob(j, &buffer, structuredOutbound)
			continue
		}

		// Under pressure, aggregate a bounded batch to reduce total writer
		// syscalls. This adaptive path only activates with substantial backlog.
		batchJobs := []*job{j}
		for i := 1; i < adaptiveBatchMaxJobs; i++ {
			select {
			case next := <-jobsChannel:
				if next.Passback != nil {
					next.Passback <- struct{}{}
					continue
				}
				batchJobs = append(batchJobs, next)
			default:
				i = adaptiveBatchMaxJobs
			}
		}

		batchData = batchData[:0]
		for i := range batchJobs {
			payload := renderLogJobPayload(batchJobs[i], &buffer, structuredOutbound)
			for w := range batchJobs[i].Writers {
				batchData = appendWriterBatch(batchData, batchJobs[i].Writers[w], payload)
			}
			jobsPool.Put(batchJobs[i])
		}

		for i := range batchData {
			n, err := batchData[i].writer.Write(batchData[i].data)
			if err != nil {
				displayError(fmt.Errorf("%T %w", batchData[i].writer, err))
			} else if n != len(batchData[i].data) {
				displayError(fmt.Errorf("%T %w", batchData[i].writer, io.ErrShortWrite))
			}
			batchData[i].data = batchData[i].data[:0]
		}
	}
}

func appendWriterBatch(batchData []writerBatch, writer io.Writer, payload []byte) []writerBatch {
	for i := range batchData {
		if batchData[i].writer == writer {
			batchData[i].data = append(batchData[i].data, payload...)
			return batchData
		}
	}
	batchData = append(batchData, writerBatch{writer: writer})
	last := len(batchData) - 1
	batchData[last].data = append(batchData[last].data, payload...)
	return batchData
}

func processSingleLogJob(j *job, buffer *[]byte, structuredOutbound ExtraFields) {
	payload := renderLogJobPayload(j, buffer, structuredOutbound)
	for x := range j.Writers {
		n, err := j.Writers[x].Write(payload)
		if err != nil {
			displayError(fmt.Errorf("%T %w", j.Writers[x], err))
		} else if n != len(payload) {
			displayError(fmt.Errorf("%T %w", j.Writers[x], io.ErrShortWrite))
		}
	}
	jobsPool.Put(j)
}

func renderLogJobPayload(j *job, buffer *[]byte, structuredOutbound ExtraFields) []byte {
	msg := j.fn()
	*buffer = (*buffer)[:0]
	if j.StructuredLogging {
		structuredOutbound[message] = msg
		structuredOutbound[timestamp] = time.Now().UnixMilli()
		structuredOutbound[severity] = j.Severity
		structuredOutbound[subLoggerName] = j.SubLoggerName
		structuredOutbound[botName] = j.Instance
		for k, v := range j.StructuredFields {
			if _, ok := structuredOutbound[k]; ok {
				displayError(fmt.Errorf("structured logging: cannot overwrite key [%s]", k))
				continue
			}
			structuredOutbound[k] = v
		}
		marshaled, err := json.Marshal(structuredOutbound)
		if err != nil {
			log.Println("log: failed to marshal structured log data:", err)
			*buffer = append(*buffer, '\n')
		} else {
			*buffer = append(*buffer, marshaled...)
			*buffer = append(*buffer, '\n')
		}
		for k := range j.StructuredFields {
			delete(structuredOutbound, k)
		}
		return *buffer
	}

	if j.Prefix != "" {
		*buffer = append(*buffer, j.Prefix...)
	} else {
		*buffer = append(*buffer, j.Header...)
		if j.ShowLogSystemName {
			*buffer = append(*buffer, j.Spacer...)
			*buffer = append(*buffer, j.SubLoggerName...)
		}
		*buffer = append(*buffer, j.Spacer...)
	}
	if j.TimestampFormat != "" {
		*buffer = time.Now().AppendFormat(*buffer, j.TimestampFormat)
	}
	*buffer = append(*buffer, j.Spacer...)
	*buffer = append(*buffer, msg...)
	if msg == "" || msg[len(msg)-1] != '\n' {
		*buffer = append(*buffer, '\n')
	}
	return *buffer
}

// deferral defines functionality that will capture data string processing and
// defer that to the worker pool if needed.
type deferral func() string

// StageLogEvent stages a new logger event in a jobs channel to be processed by
// a worker pool. This segregates the need to process the log string and the
// writes to the required io.Writer.
func (mw *multiWriterHolder) StageLogEvent(fn deferral, header, prefix, slName, spacer, timestampFormat, instance, level string, showLogSystemName, bypassWarning, structuredLog, dropDebugOnOverflow bool, fields map[Key]any) {
	newJob := jobsPool.Get().(*job) //nolint:forcetypeassert // Not necessary from a pool
	newJob.Writers = mw.writers
	newJob.fn = fn
	newJob.Header = header
	newJob.Prefix = prefix
	newJob.SubLoggerName = slName
	newJob.ShowLogSystemName = showLogSystemName
	newJob.Spacer = spacer
	newJob.TimestampFormat = timestampFormat
	newJob.Instance = instance
	newJob.StructuredFields = fields
	newJob.StructuredLogging = structuredLog
	newJob.Severity = level
	newJob.Passback = nil

	select {
	case jobsChannel <- newJob:
	default:
		if dropDebugOnOverflow && level == "debug" {
			jobsPool.Put(newJob)
			return
		}
		// This will cause temporary caller impedance, which can have a knock
		// on effect in processing.
		if !bypassWarning {
			log.Printf("Logger warning: %v\n", errJobsChannelIsFull)
		}
		jobsChannel <- newJob
	}
}

// multiWriter make and return a new copy of multiWriterHolder
func multiWriter(writers ...io.Writer) (*multiWriterHolder, error) {
	mw := &multiWriterHolder{}
	for x := range writers {
		err := mw.add(writers[x])
		if err != nil {
			return nil, err
		}
	}
	return mw, nil
}

// add appends a new writer to the multiwriter slice
func (mw *multiWriterHolder) add(writer io.Writer) error {
	if writer == nil {
		return errWriterIsNil
	}

	if slices.Contains(mw.writers, writer) {
		return errWriterAlreadyLoaded
	}

	mw.writers = append(mw.writers, writer)
	return nil
}
