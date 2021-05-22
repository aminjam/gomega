package gmeasure

import (
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/onsi/gomega/gmeasure/table"
)

type SamplingConfig struct {
	N           int
	Duration    time.Duration
	NumParallel int
}

type Units string
type Annotation string
type Style string

type PrecisionBundle struct {
	Duration    time.Duration
	ValueFormat string
}

func Precision(p interface{}) PrecisionBundle {
	out := DefaultPrecisionBundle
	switch reflect.TypeOf(p) {
	case reflect.TypeOf(time.Duration(0)):
		out.Duration = p.(time.Duration)
	case reflect.TypeOf(int(0)):
		out.ValueFormat = fmt.Sprintf("%%.%df", p.(int))
	default:
		panic("invalid precision type, must be time.Duration or int")
	}
	return out
}

var DefaultPrecisionBundle = PrecisionBundle{
	Duration:    100 * time.Microsecond,
	ValueFormat: "%.3f",
}

type extractedDecorations struct {
	annotation      Annotation
	units           Units
	precisionBundle PrecisionBundle
	style           Style
}

func extractDecorations(args []interface{}) extractedDecorations {
	var out extractedDecorations
	out.precisionBundle = DefaultPrecisionBundle

	for _, arg := range args {
		switch reflect.TypeOf(arg) {
		case reflect.TypeOf(out.annotation):
			out.annotation = arg.(Annotation)
		case reflect.TypeOf(out.units):
			out.units = arg.(Units)
		case reflect.TypeOf(out.precisionBundle):
			out.precisionBundle = arg.(PrecisionBundle)
		case reflect.TypeOf(out.style):
			out.style = arg.(Style)
		default:
			panic(fmt.Sprintf("unrecognized argument %#v", arg))
		}
	}

	return out
}

type Experiment struct {
	Name       string
	LogEntries LogEntries
	lock       *sync.Mutex
}

func NewExperiment(name string) *Experiment {
	experiment := &Experiment{
		Name: name,
		lock: &sync.Mutex{},
	}
	return experiment
}

func (e *Experiment) report(enableStyling bool) string {
	t := table.NewTable()
	t.TableStyle.EnableTextStyling = enableStyling
	t.AppendRow(table.R(
		table.C("Name"), table.C("N"), table.C("Min"), table.C("Median"), table.C("Mean"), table.C("StdDev"), table.C("Max"),
		table.Divider("="),
		"{{bold}}",
	))

	for _, entry := range e.LogEntries {
		r := table.R(entry.Style)
		t.AppendRow(r)
		switch entry.Type {
		case LogEntryTypeNote:
			r.AppendCell(table.C(entry.Note))
		case LogEntryTypeValue, LogEntryTypeDuration:
			name := entry.Name
			if entry.Units != "" {
				name += " [" + entry.Units + "]"
			}
			r.AppendCell(table.C(name))
			r.AppendCell(entry.Stats().cells()...)
		}
	}

	out := e.Name + "\n"
	if enableStyling {
		out = "{{bold}}" + out + "{{/}}"
	}
	out += t.Render()
	return out
}

func (e *Experiment) ColorableString() string {
	return e.report(true)
}

func (e *Experiment) String() string {
	return e.report(false)
}

func (e *Experiment) RecordNote(note string, args ...interface{}) {
	decorations := extractDecorations(args)

	e.lock.Lock()
	defer e.lock.Unlock()
	e.LogEntries = append(e.LogEntries, LogEntry{
		ExperimentName: e.Name,
		Type:           LogEntryTypeNote,
		Note:           note,
		Style:          string(decorations.style),
	})
}

// Recording durations
func (e *Experiment) RecordDuration(name string, duration time.Duration, args ...interface{}) {
	decorations := extractDecorations(args)
	e.recordDuration(name, duration, decorations)
}

func (e *Experiment) MeasureDuration(name string, callback func(), args ...interface{}) time.Duration {
	t := time.Now()
	callback()
	duration := time.Since(t)
	e.RecordDuration(name, duration, args...)
	return duration
}

func (e *Experiment) SampleDuration(name string, callback func(idx int), samplingConfig SamplingConfig, args ...interface{}) {
	decorations := extractDecorations(args)
	e.Sample(func(idx int) {
		t := time.Now()
		callback(idx)
		duration := time.Since(t)
		e.recordDuration(name, duration, decorations)
	}, samplingConfig)
}

func (e *Experiment) SampleAnnotatedDuration(name string, callback func(idx int) Annotation, samplingConfig SamplingConfig, args ...interface{}) {
	decorations := extractDecorations(args)
	e.Sample(func(idx int) {
		t := time.Now()
		decorations.annotation = callback(idx)
		duration := time.Since(t)
		e.recordDuration(name, duration, decorations)
	}, samplingConfig)
}

func (e *Experiment) recordDuration(name string, duration time.Duration, decorations extractedDecorations) {
	e.lock.Lock()
	defer e.lock.Unlock()
	idx := e.LogEntries.IdxWithName(name)
	if idx == -1 {
		entry := LogEntry{
			ExperimentName:  e.Name,
			Type:            LogEntryTypeDuration,
			Name:            name,
			Units:           "duration",
			Durations:       []time.Duration{duration},
			PrecisionBundle: decorations.precisionBundle,
			Style:           string(decorations.style),
			Annotations:     []string{string(decorations.annotation)},
		}
		e.LogEntries = append(e.LogEntries, entry)
	} else {
		if e.LogEntries[idx].Type != LogEntryTypeDuration {
			panic(fmt.Sprintf("attempting to record duration with name '%s'.  That name is already in-use for recording values.", name))
		}
		e.LogEntries[idx].Durations = append(e.LogEntries[idx].Durations, duration)
		e.LogEntries[idx].Annotations = append(e.LogEntries[idx].Annotations, string(decorations.annotation))
	}
}

// Stopwatch support
func (e *Experiment) NewStopwatch() *Stopwatch {
	return newStopwatch(e)
}

// Recording values
func (e *Experiment) RecordValue(name string, value float64, args ...interface{}) {
	decorations := extractDecorations(args)
	e.recordValue(name, value, decorations)
}

func (e *Experiment) MeasureValue(name string, callback func() float64, args ...interface{}) float64 {
	value := callback()
	e.RecordValue(name, value, args...)
	return value
}

func (e *Experiment) SampleValue(name string, callback func(idx int) float64, samplingConfig SamplingConfig, args ...interface{}) {
	decorations := extractDecorations(args)
	e.Sample(func(idx int) {
		value := callback(idx)
		e.recordValue(name, value, decorations)
	}, samplingConfig)
}

func (e *Experiment) SampleAnnotatedValue(name string, callback func(idx int) (float64, Annotation), samplingConfig SamplingConfig, args ...interface{}) {
	decorations := extractDecorations(args)
	e.Sample(func(idx int) {
		var value float64
		value, decorations.annotation = callback(idx)
		e.recordValue(name, value, decorations)
	}, samplingConfig)
}

func (e *Experiment) recordValue(name string, value float64, decorations extractedDecorations) {
	e.lock.Lock()
	defer e.lock.Unlock()
	idx := e.LogEntries.IdxWithName(name)
	if idx == -1 {
		entry := LogEntry{
			ExperimentName:  e.Name,
			Type:            LogEntryTypeValue,
			Name:            name,
			Style:           string(decorations.style),
			Units:           string(decorations.units),
			PrecisionBundle: decorations.precisionBundle,
			Values:          []float64{value},
			Annotations:     []string{string(decorations.annotation)},
		}
		e.LogEntries = append(e.LogEntries, entry)
	} else {
		if e.LogEntries[idx].Type != LogEntryTypeValue {
			panic(fmt.Sprintf("attempting to record value with name '%s'.  That name is already in-use for recording durations.", name))
		}
		e.LogEntries[idx].Values = append(e.LogEntries[idx].Values, value)
		e.LogEntries[idx].Annotations = append(e.LogEntries[idx].Annotations, string(decorations.annotation))
	}
}

// Sampling
func (e *Experiment) Sample(callback func(idx int), samplingConfig SamplingConfig) {
	if samplingConfig.N == 0 && samplingConfig.Duration == 0 {
		panic("you must specify at least one of SamplingConfig.N and SamplingConfig.Duration")
	}
	maxTime := time.Now().Add(100000 * time.Hour)
	if samplingConfig.Duration > 0 {
		maxTime = time.Now().Add(samplingConfig.Duration)
	}
	maxN := math.MaxInt64
	if samplingConfig.N > 0 {
		maxN = samplingConfig.N
	}
	numParallel := 1
	if samplingConfig.NumParallel > numParallel {
		numParallel = samplingConfig.NumParallel
	}

	work := make(chan int)
	if numParallel > 1 {
		for worker := 0; worker < numParallel; worker++ {
			go func() {
				for idx := range work {
					callback(idx)
				}
			}()
		}
	}

	idx := 0
	var avgDt time.Duration
	for {
		t := time.Now()
		if numParallel > 1 {
			work <- idx
		} else {
			callback(idx)
		}
		dt := time.Since(t)
		if idx >= numParallel {
			avgDt = (avgDt*time.Duration(idx-numParallel) + dt) / time.Duration(idx-numParallel+1)
		}
		idx += 1
		if idx >= maxN {
			return
		}
		if time.Now().Add(avgDt).After(maxTime) {
			return
		}
	}
}

// Fetching results
func (e *Experiment) Get(name string) LogEntry {
	e.lock.Lock()
	defer e.lock.Unlock()
	idx := e.LogEntries.IdxWithName(name)
	if idx == -1 {
		return LogEntry{}
	}
	return e.LogEntries[idx]
}

func (e *Experiment) GetStats(name string) Stats {
	entry := e.Get(name)
	e.lock.Lock()
	defer e.lock.Unlock()
	return entry.Stats()
}
