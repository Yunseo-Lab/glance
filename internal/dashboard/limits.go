package dashboard

import "errors"

const (
	MaxWidgets         = 32
	MaxBoardSources    = 32
	MaxActiveSources   = 128
	MaxEntries         = 10000
	EntryRetentionDays = 90
	MaxBoardBody       = 1 << 20
	MaxProviderBody    = 2 << 20
)

var ErrSourceLimit = errors.New("사용 중인 데이터 소스는 최대 128개입니다")

type CapacityLimits struct {
	Boards             int `json:"boards"`
	WidgetsPerBoard    int `json:"widgetsPerBoard"`
	SourcesPerBoard    int `json:"sourcesPerBoard"`
	ActiveSources      int `json:"activeSources"`
	Entries            int `json:"entries"`
	EntryRetentionDays int `json:"entryRetentionDays"`
}

func Limits() CapacityLimits {
	return CapacityLimits{MaxBoards, MaxWidgets, MaxBoardSources, MaxActiveSources, MaxEntries, EntryRetentionDays}
}
