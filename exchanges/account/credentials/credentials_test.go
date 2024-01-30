package credentials

import (
	"testing"
)

func TestIsEmpty(t *testing.T) {
	t.Parallel()
	var c *Credentials
	if !c.IsEmpty() {
		t.Fatalf("expected: %v but received: %v", true, c.IsEmpty())
	}
	c = new(Credentials)
	if !c.IsEmpty() {
		t.Fatalf("expected: %v but received: %v", true, c.IsEmpty())
	}

	c.SubAccount = "woow"
	if c.IsEmpty() {
		t.Fatalf("expected: %v but received: %v", false, c.IsEmpty())
	}
}

func TestString(t *testing.T) {
	t.Parallel()
	creds := Credentials{}
	if s := creds.String(); s != "Key:[...] SubAccount:[] ClientID:[]" {
		t.Fatal("unexpected value")
	}

	creds.Key = "12345678910111234"
	creds.SubAccount = "sub"
	creds.ClientID = "client"

	if s := creds.String(); s != "Key:[1234567891011123...] SubAccount:[sub] ClientID:[client]" {
		t.Fatal("unexpected value")
	}
}

func TestCredentialsEqual(t *testing.T) {
	t.Parallel()
	var this, that *Credentials
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this = &Credentials{}
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that = &Credentials{Key: "1337"}
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.Key = "1337"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.ClientID = "1337"
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that.ClientID = "1337"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
	this.SubAccount = "someSub"
	if this.Equal(that) {
		t.Fatal("unexpected value")
	}
	that.SubAccount = "someSub"
	if !this.Equal(that) {
		t.Fatal("unexpected value")
	}
}
