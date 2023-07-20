package test

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"math/rand"
	"net"
	"time"
)

const (
	charset = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func NewLogger() (*zap.Logger, error) {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.Encoding = "json"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("Jan 02 15:04:05.000000000")

	return loggerConfig.Build()
}

func NewSugaredLogger() (*zap.SugaredLogger, error) {
	logger, err := NewLogger()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}

func NewNamespace(name string) *apiv1.Namespace {
	namespace := apiv1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return &namespace
}

// GetRandK8sName returns a valid name for K8s objects.
func GetRandK8sName(length int) string {
	return fmt.Sprintf("name-%s", GetRandString(length))
}

var seededRand = rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec,gochecknoglobals // used in tests

// GetRandString returns a random string of the given length.
func GetRandString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

// GetFreePort determines a free port on the host. It does so by delegating the job to net.ListenTCP.
// Then providing a port of 0 to net.ListenTCP, it will automatically choose a port for us.
func GetFreePort() (int, error) {
	a, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return -1, err
	}

	var l *net.TCPListener
	l, err = net.ListenTCP("tcp", a)
	if err != nil {
		return -1, err
	}

	port := l.Addr().(*net.TCPAddr).Port
	err = l.Close()
	return port, err
}
