package filesystem

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/parquet-go/parquet-go"

	"invest/backend/internal/domain"
)

type MarketDataRepository struct {
	dataRoot string
}

type candleRow struct {
	Timestamp  time.Time `parquet:"timestamp"`
	Open       float64   `parquet:"open"`
	High       float64   `parquet:"high"`
	Low        float64   `parquet:"low"`
	Close      float64   `parquet:"close"`
	Volume     int64     `parquet:"volume"`
	Ticker     string    `parquet:"ticker"`
	Timeframe  string    `parquet:"timeframe"`
	Source     string    `parquet:"source"`
	IngestedAt time.Time `parquet:"ingested_at"`
}

type factorRow struct {
	Timestamp  time.Time `parquet:"timestamp"`
	Open       float64   `parquet:"open"`
	High       float64   `parquet:"high"`
	Low        float64   `parquet:"low"`
	Close      float64   `parquet:"close"`
	Volume     int64     `parquet:"volume"`
	Factor     string    `parquet:"factor_alias"`
	Timeframe  string    `parquet:"timeframe"`
	Source     string    `parquet:"source"`
	IngestedAt time.Time `parquet:"ingested_at"`
}

func NewMarketDataRepository(dataRoot string) *MarketDataRepository {
	return &MarketDataRepository{dataRoot: dataRoot}
}

func (r *MarketDataRepository) ListCandles(
	ctx context.Context,
	ticker string,
	timeframe string,
	from time.Time,
	to time.Time,
	limit int,
) ([]domain.Candle, error) {
	root := filepath.Join(
		r.dataRoot,
		"raw",
		"candles",
		"ticker="+ticker,
		"timeframe="+timeframe,
	)

	files, err := listPartitionFiles(root, from, to)
	if err != nil {
		return nil, err
	}

	items := make([]domain.Candle, 0)
	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		rows, err := parquet.ReadFile[candleRow](path)
		if err != nil {
			return nil, err
		}

		for _, row := range rows {
			if !timestampInRange(row.Timestamp, from, to) {
				continue
			}

			items = append(items, domain.Candle{
				Timestamp:  row.Timestamp.UTC(),
				Open:       row.Open,
				High:       row.High,
				Low:        row.Low,
				Close:      row.Close,
				Volume:     row.Volume,
				Ticker:     row.Ticker,
				Timeframe:  row.Timeframe,
				Source:     row.Source,
				IngestedAt: row.IngestedAt.UTC(),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.Before(items[j].Timestamp)
	})

	if limit > 0 && len(items) > limit {
		items = items[len(items)-limit:]
	}

	return items, nil
}

func (r *MarketDataRepository) ListFactors(
	ctx context.Context,
	timeframe string,
	from time.Time,
	to time.Time,
) ([]domain.FactorBar, error) {
	factorsRoot := filepath.Join(r.dataRoot, "raw", "factors")
	entries, err := os.ReadDir(factorsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	items := make([]domain.FactorBar, 0)
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "factor=") {
			continue
		}

		root := filepath.Join(factorsRoot, entry.Name(), "timeframe="+timeframe)
		files, err := listPartitionFiles(root, from, to)
		if err != nil {
			return nil, err
		}

		for _, path := range files {
			rows, err := parquet.ReadFile[factorRow](path)
			if err != nil {
				return nil, err
			}

			for _, row := range rows {
				if !timestampInRange(row.Timestamp, from, to) {
					continue
				}

				items = append(items, domain.FactorBar{
					Timestamp:  row.Timestamp.UTC(),
					Factor:     row.Factor,
					Open:       row.Open,
					High:       row.High,
					Low:        row.Low,
					Close:      row.Close,
					Volume:     row.Volume,
					Timeframe:  row.Timeframe,
					Source:     row.Source,
					IngestedAt: row.IngestedAt.UTC(),
				})
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Timestamp.Equal(items[j].Timestamp) {
			return items[i].Factor < items[j].Factor
		}
		return items[i].Timestamp.Before(items[j].Timestamp)
	})

	return items, nil
}

func listPartitionFiles(root string, from time.Time, to time.Time) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	startDate, hasStart := normalizeDate(from)
	endDate, hasEnd := normalizeDate(to)

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "date=") {
			continue
		}

		dateValue := strings.TrimPrefix(entry.Name(), "date=")
		partitionDate, err := time.Parse("2006-01-02", dateValue)
		if err != nil {
			continue
		}

		if hasStart && partitionDate.Before(startDate) {
			continue
		}
		if hasEnd && partitionDate.After(endDate) {
			continue
		}

		matches, err := filepath.Glob(filepath.Join(root, entry.Name(), "*.parquet"))
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}

	sort.Strings(files)
	return files, nil
}

func normalizeDate(ts time.Time) (time.Time, bool) {
	if ts.IsZero() {
		return time.Time{}, false
	}
	utc := ts.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC), true
}

func timestampInRange(ts time.Time, from time.Time, to time.Time) bool {
	utc := ts.UTC()
	if !from.IsZero() && utc.Before(from.UTC()) {
		return false
	}
	if !to.IsZero() && utc.After(to.UTC()) {
		return false
	}
	return true
}
