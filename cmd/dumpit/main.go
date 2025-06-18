package main

import (
	"encoding/binary"
	"encoding/csv"
	"flag"
	"github.com/peter-kozarec/equinox/pkg/data/mapper"
	"go.uber.org/zap"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

type Quote struct {
	Timestamp time.Time
	BidPrice  float64
	AskPrice  float64
	BidVolume float64
	AskVolume float64
}

func dumpIt(csvPath string, binFile *os.File) error {

	csvFile, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer func(csvFile *os.File) {
		_ = csvFile.Close()
	}(csvFile)

	reader := csv.NewReader(csvFile)
	var quotes []Quote

	// Skip header
	_, err = reader.Read()
	if err != nil {
		log.Fatal(err)
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// Parse timestamp
		ts, err := time.Parse("2006-01-02 15:04:05.999999999Z07:00", record[0])
		if err != nil {
			log.Fatal(err)
		}

		// Parse floats
		bidPrice, _ := strconv.ParseFloat(record[1], 64)
		askPrice, _ := strconv.ParseFloat(record[2], 64)
		bidVolume, _ := strconv.ParseFloat(record[3], 64)
		askVolume, _ := strconv.ParseFloat(record[4], 64)

		quotes = append(quotes, Quote{
			Timestamp: ts,
			BidPrice:  bidPrice,
			AskPrice:  askPrice,
			BidVolume: bidVolume,
			AskVolume: askVolume,
		})
	}

	for _, q := range quotes {
		d := mapper.BinaryTick{
			TimeStamp: q.Timestamp.UnixNano(),
			Bid:       q.BidPrice,
			Ask:       q.AskPrice,
			BidVolume: q.BidVolume,
			AskVolume: q.AskVolume,
		}
		err := binary.Write(binFile, binary.LittleEndian, d)
		if err != nil {
			log.Fatal(err)
		}
	}

	return nil
}

func dumpAll(symbol string, logger *zap.Logger) error {

	binFile, err := os.Create(symbol + ".bin")
	if err != nil {
		return err
	}
	defer func(binFile *os.File) {
		_ = binFile.Close()
	}(binFile)

	for i := 2018; i <= 2025; i++ {
		s := symbol + "_" + strconv.Itoa(i) + ".csv"
		if err := dumpIt(s, binFile); err != nil {
			return err
		}
		logger.Info("dump finished", zap.String("symbol", symbol), zap.String("file", s))
	}

	return nil
}

func main() {
	logger, _ := zap.NewProduction()
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	symbol := flag.String("symbol", "", "symbol")
	flag.Parse()

	if *symbol == "" {
		logger.Fatal("symbol is required")
	}

	if err := dumpAll(*symbol, logger); err != nil {
		logger.Fatal("Failed to dump", zap.Error(err))
	}

	logger.Info("Done dump")
}
