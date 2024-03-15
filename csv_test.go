package main

import "testing"

func Test_trimNumberPrefix(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"Test 1", args{"1. This is a test"}, "This is a test"},
		{"Test 2", args{"This is a test"}, "This is a test"},
		{"Test 3", args{" 1. This is a test"}, "This is a test"},
		{"Test 4", args{"021. This is a test"}, "This is a test"},
		{"Test 5", args{"12.This is a test"}, "This is a test"},
		{"Tab test", args{"12.	This is a test"}, "This is a test"},
		{"Tab-space-tab test", args{"52424.	 	This is a test"}, "This is a test"},
		{"no number 1", args{". This is a test"}, ". This is a test"},
		{"no number 2", args{".This is a test"}, ".This is a test"},
		/*
			{"unicode spaces all", args{"1.\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200A\u205F\u3000This is a test"}, "This is a test"},
			{"unicode spaces1", args{"1.\u00A0This is a test"}, "This is a test"},
			{"unicode spaces2", args{"1.\u180EThis is a test"}, "This is a test"},
			{"unicode spaces3", args{"1.\u2000This is a test"}, "This is a test"},
			{"unicode spaces4", args{"1.\u2001This is a test"}, "This is a test"},
			{"unicode spaces5", args{"1.\u2002This is a test"}, "This is a test"},
			{"unicode spaces6", args{"1.\u2003This is a test"}, "This is a test"},
			{"unicode spaces7", args{"1.\u2004This is a test"}, "This is a test"},
			{"unicode spaces8", args{"1.\u2005This is a test"}, "This is a test"},
			{"unicode spaces9", args{"1.\u2006This is a test"}, "This is a test"},
			{"unicode spaces10", args{"1.\u2007This is a test"}, "This is a test"},
			{"unicode spaces11", args{"1.\u2008This is a test"}, "This is a test"},
			{"unicode spaces12", args{"1.\u2009This is a test"}, "This is a test"},
			{"unicode spaces13", args{"1.\u200AThis is a test"}, "This is a test"},
			{"unicode spaces14", args{"1.\u200BThis is a test"}, "This is a test"},
			{"unicode spaces15", args{"1.\u202FThis is a test"}, "This is a test"},
			{"unicode spaces16", args{"1.\u205FThis is a test"}, "This is a test"},
			{"unicode spaces17", args{"1.\u3000This is a test"}, "This is a test"},
			{"unicode spaces18", args{"1.\uFEFFThis is a test"}, "This is a test"},
		*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := trimNumberPrefix(tt.args.s); got != tt.want {
				t.Errorf("trimNumberPrefix('%+q') = '%v', want '%v'", tt.args.s, got, tt.want)
			}
		})
	}
}
