package bagua

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestGetPrivatePorts(t *testing.T) {
	portsExceptEtcd := []int32{22, 33, 44, 55, 66}
	replicas := int32(2)
	index := strconv.FormatInt(int64(1), 10)

	pports := getPrivatePorts(portsExceptEtcd, replicas, index)
	assert.Equal(t, []int32{44, 55}, pports)
}
