package gmeasure

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/onsi/gomega/gmeasure/table"
)

type LogEntryType uint

const (
	LogEntryTypeInvalid LogEntryType = iota
	LogEntryTypeNote
	LogEntryTypeDuration
	LogEntryTypeValue
)

var letEnumSupport = newEnumSupport(map[uint]string{uint(LogEntryTypeInvalid): "INVALID LOG ENTRY TYPE", uint(LogEntryTypeNote): "Note", uint(LogEntryTypeDuration): "Duration", uint(LogEntryTypeValue): "Value"})

func (s LogEntryType) String() string { return letEnumSupport.String(uint(s)) }
func (s *LogEntryType) UnmarshalJSON(b []byte) error {
	out, err := letEnumSupport.UnmarshalJSON(b)
	*s = LogEntryType(out)
	return err
}
func (s LogEntryType) MarshalJSON() ([]byte, error) { return letEnumSupport.MarshalJSON(uint(s)) }

type LogEntry struct {
	Type LogEntryType

	ExperimentName string

	Note string

	Name            string
	Style           string
	Units           string
	PrecisionBundle PrecisionBundle

	Durations   []time.Duration
	Values      []float64
	Annotations []string
}

type LogEntries []LogEntry

func (le LogEntries) IdxWithName(name string) int {
	for idx, entry := range le {
		if entry.Name == name {
			return idx
		}
	}

	return -1
}

func (e LogEntry) report(enableStyling bool) string {
	out := ""
	style := e.Style
	if !enableStyling {
		style = ""
	}
	switch e.Type {
	case LogEntryTypeNote:
		out += fmt.Sprintf("%s - Note\n%s\n", e.ExperimentName, e.Note)
		if style != "" {
			out = style + out + "{{/}}"
		}
		return out
	case LogEntryTypeValue, LogEntryTypeDuration:
		out += fmt.Sprintf("%s - %s", e.ExperimentName, e.Name)
		if e.Units != "" {
			out += " [" + e.Units + "]"
		}
		if style != "" {
			out = style + out + "{{/}}"
		}
		out += "\n"
		out += e.Stats().String() + "\n"
	}
	t := table.NewTable()
	t.TableStyle.EnableTextStyling = enableStyling
	switch e.Type {
	case LogEntryTypeValue:
		t.AppendRow(table.R(table.C("Value", table.AlignTypeCenter), table.C("Annotation", table.AlignTypeCenter), table.Divider("="), style))
		for idx := range e.Values {
			t.AppendRow(table.R(
				table.C(fmt.Sprintf(e.PrecisionBundle.ValueFormat, e.Values[idx]), table.AlignTypeRight),
				table.C(e.Annotations[idx], "{{gray}}", table.AlignTypeLeft),
			))
		}
	case LogEntryTypeDuration:
		t.AppendRow(table.R(table.C("Duration", table.AlignTypeCenter), table.C("Annotation", table.AlignTypeCenter), table.Divider("="), style))
		for idx := range e.Durations {
			t.AppendRow(table.R(
				table.C(e.Durations[idx].Round(e.PrecisionBundle.Duration).String(), style, table.AlignTypeRight),
				table.C(e.Annotations[idx], "{{gray}}", table.AlignTypeLeft),
			))
		}
	}
	out += t.Render()
	return out
}

func (e LogEntry) ColorableString() string {
	return e.report(true)
}

func (e LogEntry) String() string {
	return e.report(false)
}

func (e LogEntry) Stats() Stats {
	if e.Type == LogEntryTypeInvalid || e.Type == LogEntryTypeNote {
		return Stats{}
	}

	out := Stats{
		ExperimentName:  e.ExperimentName,
		Name:            e.Name,
		Style:           e.Style,
		Units:           e.Units,
		PrecisionBundle: e.PrecisionBundle,
	}

	switch e.Type {
	case LogEntryTypeValue:
		out.StatsType = StatsTypeValue
		out.N = len(e.Values)
		if out.N == 0 {
			return out
		}
		indices, sum := make([]int, len(e.Values)), 0.0
		for idx, v := range e.Values {
			indices[idx] = idx
			sum += v
		}
		sort.Slice(indices, func(i, j int) bool {
			return e.Values[indices[i]] < e.Values[indices[j]]
		})
		out.ValueBundle = map[Stat]float64{
			StatMin:    e.Values[indices[0]],
			StatMax:    e.Values[indices[out.N-1]],
			StatMean:   sum / float64(out.N),
			StatStdDev: 0.0,
		}
		out.AnnotationBundle = map[Stat]string{
			StatMin: e.Annotations[indices[0]],
			StatMax: e.Annotations[indices[out.N-1]],
		}

		if out.N%2 == 0 {
			out.ValueBundle[StatMedian] = (e.Values[indices[out.N/2]] + e.Values[indices[out.N/2-1]]) / 2.0
		} else {
			out.ValueBundle[StatMedian] = e.Values[indices[(out.N-1)/2]]
		}

		for _, v := range e.Values {
			out.ValueBundle[StatStdDev] += (v - out.ValueBundle[StatMean]) * (v - out.ValueBundle[StatMean])
		}
		out.ValueBundle[StatStdDev] = math.Sqrt(out.ValueBundle[StatStdDev] / float64(out.N))
	case LogEntryTypeDuration:
		out.StatsType = StatsTypeDuration
		out.N = len(e.Durations)
		if out.N == 0 {
			return out
		}
		indices, sum := make([]int, len(e.Durations)), time.Duration(0)
		for idx, v := range e.Durations {
			indices[idx] = idx
			sum += v
		}
		sort.Slice(indices, func(i, j int) bool {
			return e.Durations[indices[i]] < e.Durations[indices[j]]
		})
		out.DurationBundle = map[Stat]time.Duration{
			StatMin:  e.Durations[indices[0]],
			StatMax:  e.Durations[indices[out.N-1]],
			StatMean: sum / time.Duration(out.N),
		}
		out.AnnotationBundle = map[Stat]string{
			StatMin: e.Annotations[indices[0]],
			StatMax: e.Annotations[indices[out.N-1]],
		}

		if out.N%2 == 0 {
			out.DurationBundle[StatMedian] = (e.Durations[indices[out.N/2]] + e.Durations[indices[out.N/2-1]]) / 2
		} else {
			out.DurationBundle[StatMedian] = e.Durations[indices[(out.N-1)/2]]
		}
		stdDev := 0.0
		for _, v := range e.Durations {
			stdDev += float64(v-out.DurationBundle[StatMean]) * float64(v-out.DurationBundle[StatMean])
		}
		out.DurationBundle[StatStdDev] = time.Duration(math.Sqrt(stdDev / float64(out.N)))
	}

	return out
}
