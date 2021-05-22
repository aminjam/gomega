package gmeasure

import "time"

type Stopwatch struct {
	Experiment    *Experiment
	t             time.Time
	pauseT        time.Time
	pauseDuration time.Duration
	running       bool
}

func newStopwatch(experiment *Experiment) *Stopwatch {
	return &Stopwatch{
		Experiment: experiment,
		t:          time.Now(),
		running:    true,
	}
}

func (s *Stopwatch) NewStopwatch() *Stopwatch {
	return newStopwatch(s.Experiment)
}

func (s *Stopwatch) Record(name string, args ...interface{}) *Stopwatch {
	if !s.running {
		panic("stopwatch is not running - call Resume or Reset before calling Record")
	}
	duration := time.Since(s.t) - s.pauseDuration
	s.Experiment.RecordDuration(name, duration, args...)
	return s
}

func (s *Stopwatch) Reset() *Stopwatch {
	s.running = true
	s.t = time.Now()
	s.pauseDuration = 0
	return s
}

func (s *Stopwatch) Pause() *Stopwatch {
	if !s.running {
		panic("stopwatch is not running - call Resume or Reset before calling Pause")
	}
	s.running = false
	s.pauseT = time.Now()
	return s
}

func (s *Stopwatch) Resume() *Stopwatch {
	if s.running {
		panic("stopwatch is running - call Pause before calling Resume")
	}
	s.running = true
	s.pauseDuration = s.pauseDuration + time.Since(s.pauseT)
	return s
}
