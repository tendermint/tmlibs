package log_test

import (
	"bytes"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/go-logfmt/logfmt"
	"github.com/tendermint/tmlibs/log"

	"github.com/stretchr/testify/require"
)

func TestLoggerLogsItsErrors(t *testing.T) {
	var buf bytes.Buffer

	logger := log.NewTMLogger(&buf)
	logger.Info("foo", "baz baz", "bar")
	msg := strings.TrimSpace(buf.String())
	if !strings.Contains(msg, logfmt.ErrInvalidKey.Error()) {
		t.Errorf("Expected logger msg to contain ErrInvalidKey, got %s", msg)
	}
}

func TestLoggerLogsWithAndWithoutTimePrefix(t *testing.T) {
	bufNoTimePrefix := new(bytes.Buffer)
	loggerNoTimePrefix := log.NewTMLogger(bufNoTimePrefix, log.OptionDisableTimePrefix)
	loggerNoTimePrefix.Info("Tendermint", "tmlibs", "bonjour")

	bufTimePrefix := new(bytes.Buffer)
	loggerTimePrefix := log.NewTMLogger(bufTimePrefix)
	loggerTimePrefix.Info("Tendermint", "tmlibs", "bonjour")

	strWithTimePrefix := strings.Replace(bufTimePrefix.String(), "\n", "", -1)
	strWithoutTimePrefix := strings.Replace(bufNoTimePrefix.String(), "\n", "", -1)
	// 1. With time prefix and without should NOT be the same.
	require.NotEqual(t, strWithTimePrefix, strWithoutTimePrefix, "they cannot be equal")

	// 2. Both should have the suffix "tmlibs=bonjour"
	require.True(t, strings.HasSuffix(strWithTimePrefix, "tmlibs=bonjour"))
	require.True(t, strings.HasSuffix(strWithoutTimePrefix, "tmlibs=bonjour"))

	// 3. Only withoutTimePrefix should have the prefix "Tendermint"
	require.True(t, strings.HasPrefix(strWithoutTimePrefix, "Tendermint"))
	require.False(t, strings.HasPrefix(strWithTimePrefix, "Tendermint"))
	// 3.1. but withTimePrefix should have prefix ".+["
	require.True(t, strings.HasPrefix(strWithTimePrefix[1:], "["))

	// 4. We should be able to parse out a time from withTimePrefix
	lBrace := strings.Index(strWithTimePrefix, "[")
	rBrace := strings.Index(strWithTimePrefix, "]")
	timeStr := strWithTimePrefix[lBrace+1 : rBrace]
	wantFmt := "01-02|15:04:05.000"
	require.Equal(t, len(timeStr), len(wantFmt))
}

func BenchmarkTMLoggerSimple(b *testing.B) {
	benchmarkRunner(b, log.NewTMLogger(ioutil.Discard), baseInfoMessage)
}

func BenchmarkTMLoggerContextual(b *testing.B) {
	benchmarkRunner(b, log.NewTMLogger(ioutil.Discard), withInfoMessage)
}

func benchmarkRunner(b *testing.B, logger log.Logger, f func(log.Logger)) {
	lc := logger.With("common_key", "common_value")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f(lc)
	}
}

var (
	baseInfoMessage = func(logger log.Logger) { logger.Info("foo_message", "foo_key", "foo_value") }
	withInfoMessage = func(logger log.Logger) { logger.With("a", "b").Info("c", "d", "f") }
)
