// Code generated by protoc-gen-go-primo. DO NOT EDIT.
// versions:
// 	protoc-gen-go-primo v
// 	protoc              v4.24.4
// source: github.com/dogmatiq/veracity/internal/eventstream/internal/journalpb/record.proto

package journalpb

import envelopepb "github.com/dogmatiq/enginekit/protobuf/envelopepb"

// Switch_Record_Operation dispatches to one of the given functions based on
// which value of the [Record] message's "Operation" one-of group is populated.
//
// It panics if x.Operation field is nil; otherwise, it returns the value
// returned by the called function. If no return value is required, use a return
// type of [error] and always return nil.
func Switch_Record_Operation[T any](
	x *Record,
	caseAppend func(*AppendOperation) T,
) T {
	switch v := x.Operation.(type) {
	case *Record_Append:
		return caseAppend(v.Append)
	default:
		panic("Switch_Record_Operation: x.Operation is nil")
	}
}

// SetStreamOffsetBefore sets the x.StreamOffsetBefore field to v.
func (x *Record) SetStreamOffsetBefore(v uint64) {
	x.StreamOffsetBefore = v
}

// SetStreamOffsetAfter sets the x.StreamOffsetAfter field to v.
func (x *Record) SetStreamOffsetAfter(v uint64) {
	x.StreamOffsetAfter = v
}

// SetAppend sets the x.Operation field to a [Operation] value containing v
func (x *Record) SetAppend(v *AppendOperation) {
	x.Operation = &Record_Append{Append: v}
}

// SetEvents sets the x.Events field to v.
func (x *AppendOperation) SetEvents(v []*envelopepb.Envelope) {
	x.Events = v
}
