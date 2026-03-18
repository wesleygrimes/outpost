package store

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	outpostv1 "github.com/wesgrimes/outpost/gen/outpost/v1"
)

// RunToProto converts a store Run to a proto Run.
func RunToProto(r *Run) *outpostv1.Run {
	if r == nil {
		return nil
	}

	pr := &outpostv1.Run{
		Id:         r.ID,
		Name:       r.Name,
		Mode:       ModeToProto(r.Mode),
		Status:     StatusToProto(r.Status),
		BaseSha:    r.BaseSHA,
		FinalSha:   r.FinalSHA,
		CreatedAt:  timestamppb.New(r.CreatedAt),
		Attach:     r.Attach,
		LogTail:    r.LogTail,
		PatchReady: r.PatchReady,
		Branch:     r.Branch,
		MaxTurns:   int32(r.MaxTurns),
		Subdir:     r.Subdir,
	}
	if r.FinishedAt != nil {
		pr.FinishedAt = timestamppb.New(*r.FinishedAt)
	}
	return pr
}

// ProtoToRun converts a proto Run to a store Run.
func ProtoToRun(pr *outpostv1.Run) *Run {
	if pr == nil {
		return nil
	}
	r := &Run{
		ID:         pr.GetId(),
		Name:       pr.GetName(),
		Mode:       ModeFromProto(pr.GetMode()),
		Status:     StatusFromProto(pr.GetStatus()),
		BaseSHA:    pr.GetBaseSha(),
		FinalSHA:   pr.GetFinalSha(),
		Attach:     pr.GetAttach(),
		LogTail:    pr.GetLogTail(),
		PatchReady: pr.GetPatchReady(),
		Branch:     pr.GetBranch(),
		MaxTurns:   int(pr.GetMaxTurns()),
		Subdir:     pr.GetSubdir(),
	}
	if pr.GetCreatedAt() != nil {
		r.CreatedAt = pr.GetCreatedAt().AsTime()
	}
	if pr.GetFinishedAt() != nil {
		t := pr.GetFinishedAt().AsTime()
		r.FinishedAt = &t
	}
	return r
}

// StatusToProto converts a store Status to a proto RunStatus.
func StatusToProto(s Status) outpostv1.RunStatus {
	switch s {
	case StatusPending:
		return outpostv1.RunStatus_RUN_STATUS_PENDING
	case StatusRunning:
		return outpostv1.RunStatus_RUN_STATUS_RUNNING
	case StatusComplete:
		return outpostv1.RunStatus_RUN_STATUS_COMPLETE
	case StatusFailed:
		return outpostv1.RunStatus_RUN_STATUS_FAILED
	case StatusDropped:
		return outpostv1.RunStatus_RUN_STATUS_DROPPED
	default:
		return outpostv1.RunStatus_RUN_STATUS_UNSPECIFIED
	}
}

// StatusFromProto converts a proto RunStatus to a store Status.
func StatusFromProto(s outpostv1.RunStatus) Status {
	switch s {
	case outpostv1.RunStatus_RUN_STATUS_PENDING:
		return StatusPending
	case outpostv1.RunStatus_RUN_STATUS_RUNNING:
		return StatusRunning
	case outpostv1.RunStatus_RUN_STATUS_COMPLETE:
		return StatusComplete
	case outpostv1.RunStatus_RUN_STATUS_FAILED:
		return StatusFailed
	case outpostv1.RunStatus_RUN_STATUS_DROPPED:
		return StatusDropped
	case outpostv1.RunStatus_RUN_STATUS_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}

// ModeToProto converts a store Mode to a proto RunMode.
func ModeToProto(m Mode) outpostv1.RunMode {
	switch m {
	case ModeInteractive:
		return outpostv1.RunMode_RUN_MODE_INTERACTIVE
	case ModeHeadless:
		return outpostv1.RunMode_RUN_MODE_HEADLESS
	default:
		return outpostv1.RunMode_RUN_MODE_UNSPECIFIED
	}
}

// ModeFromProto converts a proto RunMode to a store Mode.
func ModeFromProto(m outpostv1.RunMode) Mode {
	switch m {
	case outpostv1.RunMode_RUN_MODE_INTERACTIVE:
		return ModeInteractive
	case outpostv1.RunMode_RUN_MODE_HEADLESS:
		return ModeHeadless
	case outpostv1.RunMode_RUN_MODE_UNSPECIFIED:
		return ""
	default:
		return ""
	}
}

// TimestampToTime converts a protobuf Timestamp to a *time.Time.
func TimestampToTime(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}
