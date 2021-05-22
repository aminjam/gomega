package gmeasure

import (
	"fmt"
	"time"

	"github.com/onsi/gomega/gmeasure/table"
)

type Stat uint

const (
	StatInvalid Stat = iota
	StatMin
	StatMax
	StatMean
	StatMedian
	StatStdDev
)

var statEnumSupport = newEnumSupport(map[uint]string{uint(StatInvalid): "INVALID STAT", uint(StatMin): "Min", uint(StatMax): "Max", uint(StatMean): "Mean", uint(StatMedian): "Median", uint(StatStdDev): "StdDev"})

func (s Stat) String() string { return statEnumSupport.String(uint(s)) }
func (s *Stat) UnmarshalJSON(b []byte) error {
	out, err := statEnumSupport.UnmarshalJSON(b)
	*s = Stat(out)
	return err
}
func (s Stat) MarshalJSON() ([]byte, error) { return statEnumSupport.MarshalJSON(uint(s)) }

type StatsType uint

const (
	StatsTypeInvalid StatsType = iota
	StatsTypeValue
	StatsTypeDuration
)

var statsTypeEnumSupport = newEnumSupport(map[uint]string{uint(StatsTypeInvalid): "INVALID STATS TYPE", uint(StatsTypeValue): "StatsTypeValue", uint(StatsTypeDuration): "StatsTypeDuration"})

func (s StatsType) String() string { return statsTypeEnumSupport.String(uint(s)) }
func (s *StatsType) UnmarshalJSON(b []byte) error {
	out, err := statsTypeEnumSupport.UnmarshalJSON(b)
	*s = StatsType(out)
	return err
}
func (s StatsType) MarshalJSON() ([]byte, error) { return statsTypeEnumSupport.MarshalJSON(uint(s)) }

type Stats struct {
	StatsType StatsType

	ExperimentName string
	Name           string
	Units          string
	Style          string

	N int

	PrecisionBundle  PrecisionBundle
	ValueBundle      map[Stat]float64
	DurationBundle   map[Stat]time.Duration
	AnnotationBundle map[Stat]string
}

func (s Stats) String() string {
	return fmt.Sprintf("%s < [%s] | <%s> ±%s < %s", s.StringFor(StatMin), s.StringFor(StatMedian), s.StringFor(StatMean), s.StringFor(StatStdDev), s.StringFor(StatMax))
}

func (s Stats) MeanStdDevCharacterization() string {
	return fmt.Sprintf("<%s> ±%s", s.StringFor(StatMean), s.StringFor(StatStdDev))
}

func (s Stats) ValueFor(stat Stat) float64 {
	return s.ValueBundle[stat]
}

func (s Stats) DurationFor(stat Stat) time.Duration {
	return s.DurationBundle[stat]
}

func (s Stats) FloatFor(stat Stat) float64 {
	switch s.StatsType {
	case StatsTypeValue:
		return s.ValueFor(stat)
	case StatsTypeDuration:
		return float64(s.DurationFor(stat))
	}
	return 0
}

func (s Stats) StringFor(stat Stat) string {
	switch s.StatsType {
	case StatsTypeValue:
		return fmt.Sprintf(s.PrecisionBundle.ValueFormat, s.ValueFor(stat))
	case StatsTypeDuration:
		return s.DurationFor(stat).Round(s.PrecisionBundle.Duration).String()
	}
	return ""
}

func (s Stats) cells() []table.Cell {
	out := []table.Cell{}
	out = append(out, table.C(fmt.Sprintf("%d", s.N)))
	for _, stat := range []Stat{StatMin, StatMedian, StatMean, StatStdDev, StatMax} {
		content := s.StringFor(stat)
		if s.AnnotationBundle[stat] != "" {
			content += "\n" + s.AnnotationBundle[stat]
		}
		out = append(out, table.C(content))
	}
	return out
}
