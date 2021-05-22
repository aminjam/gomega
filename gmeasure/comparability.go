package gmeasure

import (
	"fmt"
	"math"

	"github.com/onsi/gomega/format"
)

func ComputeStatsZScore(a, b Stats) (float64, error) {
	if a.StatsType != b.StatsType {
		return 0, fmt.Errorf("Stats type mismatch - have %s and %s which are not the same", a.StatsType, b.StatsType)
	}
	if a.N == 0 || b.N == 0 {
		return 0, fmt.Errorf("At least one set of Stats is empty")
	}

	µA := a.FloatFor(StatMean)
	σA := a.FloatFor(StatStdDev) / math.Sqrt(float64(a.N))
	µB := b.FloatFor(StatMean)
	σB := b.FloatFor(StatStdDev) / math.Sqrt(float64(b.N))

	return math.Abs(µA-µB) / math.Sqrt(σA*σA+σB*σB), nil
}

/*
BeComparableTo uses a Z-score test (http://homework.uoregon.edu/pub/class/es202/ztest.html) to compare two stats distributions.  It returns a match if the two distributions are deemed comparable.
Both expected and actual must be stat distributions (either both ValueStats or DurationStats).

The Z-score that is computed is a scale-free measure of the distance between the means of the distributions.  It assumes a normal distribution and may not be reliable for heavily skewed distributions.
By default a cut-off of Z <= 2 is used to deremine if the distributions are comparable.  You can adjust this cutoff by passing in a float64 to the matcher.  Higher values are more lenient, lower values are more stringent.
*/
func BeComparableTo(stats Stats, zCutoff ...float64) *BeComparableToMatcher {
	out := &BeComparableToMatcher{
		zCutoff:  2.0,
		expected: stats,
	}
	if len(zCutoff) > 0 {
		out.zCutoff = zCutoff[0]
	}
	return out
}

type BeComparableToMatcher struct {
	actual   Stats
	expected Stats
	z        float64
	zCutoff  float64
}

func (matcher *BeComparableToMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Actual Distribution: %s\nExpectedDistribution: %s\nZ-score: %.3f\nDistributions are not comparable as Z-score is > Z-score cutoff (%.3f).", matcher.actual.MeanStdDevCharacterization(), matcher.expected.MeanStdDevCharacterization(), matcher.z, matcher.zCutoff)
}

func (matcher *BeComparableToMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Actual Distribution: %s\nExpectedDistribution: %s\nZ-score: %.3f\nDistributions are comparable as Z-score is <= Z-score cutoff (%.3f).", matcher.actual.MeanStdDevCharacterization(), matcher.expected.MeanStdDevCharacterization(), matcher.z, matcher.zCutoff)
}

func (matcher *BeComparableToMatcher) Match(actualIface interface{}) (bool, error) {
	var ok bool
	var err error
	matcher.actual, ok = actualIface.(Stats)
	if !ok {
		return false, fmt.Errorf("Actual value must be of type Stats.  Got:\n%s", format.Object(actualIface, 1))
	}
	matcher.z, err = ComputeStatsZScore(matcher.actual, matcher.expected)
	if err != nil {
		return false, err
	}
	return matcher.z <= matcher.zCutoff, nil
}
