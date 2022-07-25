package aggregate_test

import (
	"context"

	. "github.com/dogmatiq/veracity/aggregate"
	. "github.com/jmalloc/gomegax"
	. "github.com/onsi/gomega"
)

// revisionReaderStub is a test implementation of the RevisionReader interface.
type revisionReaderStub struct {
	RevisionReader

	ReadBoundsFunc func(
		ctx context.Context,
		hk, id string,
	) (Bounds, error)

	ReadRevisionsFunc func(
		ctx context.Context,
		hk, id string,
		begin uint64,
	) (revisions []Revision, _ error)
}

func (s *revisionReaderStub) ReadBounds(
	ctx context.Context,
	hk, id string,
) (Bounds, error) {
	if s.ReadBoundsFunc != nil {
		return s.ReadBoundsFunc(ctx, hk, id)
	}

	if s.RevisionReader != nil {
		return s.RevisionReader.ReadBounds(ctx, hk, id)
	}

	return Bounds{}, nil
}

func (s *revisionReaderStub) ReadRevisions(
	ctx context.Context,
	hk, id string,
	begin uint64,
) (revisions []Revision, _ error) {
	if s.ReadRevisionsFunc != nil {
		return s.ReadRevisionsFunc(ctx, hk, id, begin)
	}

	if s.RevisionReader != nil {
		return s.RevisionReader.ReadRevisions(ctx, hk, id, begin)
	}

	return nil, nil
}

// revisionWriterStub is a test implementation of the RevisionWriter interface.
type revisionWriterStub struct {
	RevisionWriter

	PrepareRevisionFunc func(
		ctx context.Context,
		hk, id string,
		rev Revision,
	) error

	CommitRevisionFunc func(
		ctx context.Context,
		hk, id string,
		rev Revision,
	) error
}

func (s *revisionWriterStub) PrepareRevision(
	ctx context.Context,
	hk, id string,
	rev Revision,
) error {
	if s.PrepareRevisionFunc != nil {
		return s.PrepareRevisionFunc(ctx, hk, id, rev)
	}

	if s.RevisionWriter != nil {
		return s.RevisionWriter.PrepareRevision(ctx, hk, id, rev)
	}

	return nil
}

func (s *revisionWriterStub) CommitRevision(
	ctx context.Context,
	hk, id string,
	rev Revision,
) error {
	if s.CommitRevisionFunc != nil {
		return s.CommitRevisionFunc(ctx, hk, id, rev)
	}

	if s.RevisionWriter != nil {
		return s.RevisionWriter.CommitRevision(ctx, hk, id, rev)
	}

	return nil
}

// expectRevisions reads all revisions starting from begin and asserts that they
// are equal to expected.
func expectRevisions(
	ctx context.Context,
	reader RevisionReader,
	hk, id string,
	begin uint64,
	expected []Revision,
) {
	var actual []Revision

	for {
		revisions, err := reader.ReadRevisions(
			ctx,
			hk,
			id,
			begin,
		)
		ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

		if len(revisions) == 0 {
			break
		}

		actual = append(actual, revisions...)
		begin += uint64(len(revisions))
	}

	if len(actual) == 0 && len(expected) == 0 {
		return
	}

	ExpectWithOffset(1, actual).To(EqualX(expected))
}

// commitRevision prepares and immediately commits a revision.
func commitRevision(
	ctx context.Context,
	writer RevisionWriter,
	hk, id string,
	rev Revision,
) {
	err := writer.PrepareRevision(ctx, hk, id, rev)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())

	err = writer.CommitRevision(ctx, hk, id, rev)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
}
