package chart

import (
	"math"
	"testing"

	"github.com/croaky/is"
)

// A count keeps its digits where they are still countable and compacts
// where they are not. Money always compacts.
func TestFormatValueCountAndMoney(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(FormatValue(0, Count), "0")
	is.Eq(FormatValue(8_246, Count), "8,246")
	is.Eq(FormatValue(92_500, Count), "92,500")
	is.Eq(FormatValue(999_960, Count), "999,960")
	is.Eq(FormatValue(2_000_000, Count), "2M")
	is.Eq(FormatValue(2_400_000, Count), "2.4M")
	is.Eq(FormatValue(-500, Count), "-500")
	is.Eq(FormatValue(-1_234_567, Count), "-1.2M")

	is.Eq(FormatValue(0, USD), "$0")
	is.Eq(FormatValue(750, USD), "$750")
	is.Eq(FormatValue(8_246, USD), "$8.2K")
	is.Eq(FormatValue(40_000_000, USD), "$40M")
	is.Eq(FormatValue(500_000_000, USD), "$500M")
	is.Eq(FormatValue(1_500_000_000, USD), "$1.5B")
	is.Eq(FormatValue(2_500_000_000_000, USD), "$2.5T")
}

// A value that rounds up to 1,000 of one unit prints in the next unit.
func TestFormatValueCompactCarries(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(FormatValue(999_960, USD), "$1M")
	is.Eq(FormatValue(999_960_000, USD), "$1B")
	is.Eq(FormatValue(999_400, USD), "$999.4K")
}

func TestFormatValuePrice(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(FormatValue(0, Price), "$0")
	is.Eq(FormatValue(0.007377, Price), "$0.00738")
	is.Eq(FormatValue(0.129, Price), "$0.129")
	is.Eq(FormatValue(1.32, Price), "$1.32")
	is.Eq(FormatValue(12.54, Price), "$12.5")
	is.Eq(FormatValue(1234.5, Price), "$1,235")
}

// A share below 1% keeps a decimal, so it does not print as 0%. A share
// that rounds to zero or to one prints as a whole number.
func TestFormatValuePercent(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(FormatValue(7.6, Percent), "8%")
	is.Eq(FormatValue(124, Percent), "124%")
	is.Eq(FormatValue(0.4, Percent), "0.4%")
	is.Eq(FormatValue(0.04, Percent), "0%")
	is.Eq(FormatValue(0.96, Percent), "1%")
	is.Eq(FormatValue(-0.4, Percent), "-0.4%")
}

func TestFormatValueMillis(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(FormatValue(0, Millis), "0ms")
	is.Eq(FormatValue(850, Millis), "850ms")
	is.Eq(FormatValue(999.6, Millis), "1s")
	is.Eq(FormatValue(1000, Millis), "1s")
	is.Eq(FormatValue(1200, Millis), "1.2s")
	is.Eq(FormatValue(15_000, Millis), "15s")
	is.Eq(FormatValue(59_990, Millis), "1m")
	is.Eq(FormatValue(90_000, Millis), "1.5m")
	is.Eq(FormatValue(3_599_000, Millis), "1h")
	is.Eq(FormatValue(3_600_000, Millis), "1h")
	is.Eq(FormatValue(7_200_000, Millis), "2h")
	is.Eq(FormatValue(-1500, Millis), "-1.5s")
}

// A value that rounds to zero prints as zero, never as -0, and a value
// that is not a number prints as a dash in place of NaN.
func TestFormatValueDegenerate(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(FormatValue(-0.0001, Count), "0")
	is.Eq(FormatValue(-0.0001, USD), "$0")
	is.Eq(FormatValue(-0.0001, Millis), "0ms")
	is.Eq(FormatValue(-0.01, Percent), "0%")
	is.Eq(FormatValue(math.NaN(), Count), "–")
	is.Eq(FormatValue(math.Inf(1), USD), "–")
}

// A signed value prints its sign, and zero prints none.
func TestFormatValueSigned(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(FormatValue(6, Signed), "+6")
	is.Eq(FormatValue(-6, Signed), "−6")
	is.Eq(FormatValue(0, Signed), "0")
	is.Eq(FormatValue(-0.2, Signed), "0")
	is.Eq(FormatValue(12_345, Signed), "+12,345")
	is.Eq(NewValueAxis(-6, 6, Signed).Labels(), []string{"+6", "+4", "+2", "0", "−2", "−4", "−6"})
}

func TestPluralize(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.Eq(pluralize(1, "point", "points"), "1 point")
	is.Eq(pluralize(2, "point", "points"), "2 points")
	is.Eq(pluralize(1200, "point", "points"), "1,200 points")
}

func TestDelimited(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(delimited(0), "0")
	is.Eq(delimited(999), "999")
	is.Eq(delimited(1000), "1,000")
	is.Eq(delimited(123456), "123,456")
	is.Eq(delimited(-1234567), "-1,234,567")
}
