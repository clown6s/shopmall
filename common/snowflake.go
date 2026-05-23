package common

import (
	"errors"
	"sync"
	"time"
)

// Snowflake 雪花算法 ID 生成器
type Snowflake struct {
	mu         sync.Mutex
	lastTime   int64
	workerID   int64
	sequence   int64
	workerBits uint8
	seqBits    uint8
}

var (
	ErrClockMovedBackward = errors.New("clock moved backward")
	instance              *Snowflake
	once                  sync.Once
)

// NewSnowflake 创建雪花 ID 生成器（workerID: 机器ID，0-31）
func NewSnowflake(workerID int64) *Snowflake {
	if workerID < 0 || workerID > 31 {
		workerID = workerID & 0x1F
	}
	return &Snowflake{
		lastTime:   0,
		workerID:   workerID,
		sequence:   0,
		workerBits: 5,
		seqBits:    12,
	}
}

// InitSnowflake 全局初始化
func InitSnowflake(workerID int64) {
	once.Do(func() {
		instance = NewSnowflake(workerID)
	})
}

// NextID 生成下一个 ID
func (s *Snowflake) NextID() (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	epoch := int64(1704067200000) // 2024-01-01

	if now < s.lastTime {
		offset := s.lastTime - now
		if offset > 5000 {
			return 0, ErrClockMovedBackward
		}
		time.Sleep(time.Duration(offset) * time.Millisecond)
		now = time.Now().UnixMilli()
		if now < s.lastTime {
			return 0, ErrClockMovedBackward
		}
	}

	if now == s.lastTime {
		s.sequence = (s.sequence + 1) & ((1 << s.seqBits) - 1)
		if s.sequence == 0 {
			for now <= s.lastTime {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}

	s.lastTime = now

	shift := s.seqBits
	return ((now - epoch) << (s.workerBits + shift)) |
		(s.workerID << shift) |
		s.sequence, nil
}

// NextBrandID 生成品牌 ID
func NextBrandID() (int64, error) {
	if instance == nil {
		InitSnowflake(1)
	}
	return instance.NextID()
}
