package service

import (
	"math"

	"invest/backend/internal/domain"
)

type IndicatorValue struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

func CalculateEMA(candles []domain.Candle, period int) []IndicatorValue {
	if len(candles) < period {
		return nil
	}
	out := make([]IndicatorValue, 0, len(candles))
	k := 2.0 / float64(period+1)
	
	// Start with SMA for the first EMA value
	var sum float64
	for i := 0; i < period; i++ {
		sum += candles[i].Close
	}
	prevEMA := sum / float64(period)
	out = append(out, IndicatorValue{
		Timestamp: candles[period-1].Timestamp.Format(domain.RFC3339Millis),
		Value:     prevEMA,
	})

	for i := period; i < len(candles); i++ {
		ema := (candles[i].Close-prevEMA)*k + prevEMA
		out = append(out, IndicatorValue{
			Timestamp: candles[i].Timestamp.Format(domain.RFC3339Millis),
			Value:     ema,
		})
		prevEMA = ema
	}
	return out
}

func CalculateRSI(candles []domain.Candle, period int) []IndicatorValue {
	if len(candles) <= period {
		return nil
	}
	out := make([]IndicatorValue, 0, len(candles))

	var avgGain, avgLoss float64
	for i := 1; i <= period; i++ {
		change := candles[i].Close - candles[i-1].Close
		if change > 0 {
			avgGain += change
		} else {
			avgLoss -= change
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	for i := period; i < len(candles); i++ {
		if i > period {
			change := candles[i].Close - candles[i-1].Close
			var gain, loss float64
			if change > 0 {
				gain = change
			} else {
				loss = -change
			}
			avgGain = (avgGain*float64(period-1) + gain) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		}

		rs := 100.0
		if avgLoss != 0 {
			rs = avgGain / avgLoss
		}
		rsi := 100.0 - (100.0 / (1.0 + rs))
		out = append(out, IndicatorValue{
			Timestamp: candles[i].Timestamp.Format(domain.RFC3339Millis),
			Value:     rsi,
		})
	}
	return out
}

func CalculateATR(candles []domain.Candle, period int) []IndicatorValue {
	if len(candles) <= period {
		return nil
	}
	out := make([]IndicatorValue, 0, len(candles))

	tr := make([]float64, len(candles))
	for i := 1; i < len(candles); i++ {
		h := candles[i].High
		l := candles[i].Low
		pc := candles[i-1].Close
		tr[i] = math.Max(h-l, math.Max(math.Abs(h-pc), math.Abs(l-pc)))
	}

	var sum float64
	for i := 1; i <= period; i++ {
		sum += tr[i]
	}
	prevATR := sum / float64(period)
	out = append(out, IndicatorValue{
		Timestamp: candles[period].Timestamp.Format(domain.RFC3339Millis),
		Value:     prevATR,
	})

	for i := period + 1; i < len(candles); i++ {
		atr := (prevATR*float64(period-1) + tr[i]) / float64(period)
		out = append(out, IndicatorValue{
			Timestamp: candles[i].Timestamp.Format(domain.RFC3339Millis),
			Value:     atr,
		})
		prevATR = atr
	}
	return out
}
