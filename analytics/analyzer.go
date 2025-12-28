package analytics

import (
	"math"
	"sync"
)

type Analyzer struct {
	windowSize int          // размер окна (50)
	values     []float64    // кольцевой буфер значений
	mu         sync.RWMutex // для конкурентного доступа
	index      int          // текущая позиция в буфере
	isFull     bool         // буфер заполнен?

	// Текущие расчетные значения
	currentAvg float64
	currentStd float64
	anomalies  int // счетчик аномалий
}

func NewAnalyzer(windowSize int) *Analyzer {
	return &Analyzer{
		windowSize: windowSize,
		values:     make([]float64, windowSize),
		index:      0,
		isFull:     false,
		currentAvg: 0,
		currentStd: 0,
		anomalies:  0,
	}
}

// AddValue добавляет новое значение и пересчитывает статистику
func (a *Analyzer) AddValue(value float64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Добавляем значение в кольцевой буфер
	a.values[a.index] = value
	a.index++

	// Проверяем, заполнен ли буфер
	if a.index >= a.windowSize {
		a.index = 0
		a.isFull = true
	}

	// Пересчитываем статистику
	a.recalculate()

	// Проверяем на аномалию (только если буфер заполнен)
	if a.isFull {
		return a.isAnomaly(value)
	}
	return false
}

// recalculate вычисляет среднее и стандартное отклонение
func (a *Analyzer) recalculate() {
	size := a.windowSize
	if !a.isFull {
		if a.index == 0 {
			a.currentAvg = 0
			a.currentStd = 0
			return
		}
		size = a.index
	}

	// 1. Вычисляем среднее
	sum := 0.0
	for i := 0; i < size; i++ {
		sum += a.values[i]
	}
	avg := sum / float64(size)

	// 2. Вычисляем стандартное отклонение
	variance := 0.0
	for i := 0; i < size; i++ {
		diff := a.values[i] - avg
		variance += diff * diff
	}
	std := math.Sqrt(variance / float64(size))

	a.currentAvg = avg
	a.currentStd = std
}

// isAnomaly проверяет, является ли значение аномалией (> 2σ)
func (a *Analyzer) isAnomaly(value float64) bool {
	if a.currentStd == 0 {
		return false // нет отклонения = нет аномалий
	}

	// Вычисляем z-score
	zScore := math.Abs(value-a.currentAvg) / a.currentStd

	if zScore > 2.0 { // порог 2σ
		a.anomalies++
		return true
	}
	return false
}

// Методы для получения текущих значений (thread-safe)
func (a *Analyzer) GetCurrentAvg() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentAvg
}

func (a *Analyzer) GetCurrentStd() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.currentStd
}

func (a *Analyzer) GetAnomaliesCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.anomalies
}

func (a *Analyzer) GetWindowSize() int {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.isFull {
		return a.windowSize
	}
	return a.index
}

// GetValues возвращает копию текущего окна (для отладки)
func (a *Analyzer) GetValues() []float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()

	size := a.windowSize
	if !a.isFull && a.index > 0 {
		size = a.index
	}

	values := make([]float64, size)
	copy(values, a.values[:size])
	return values
}
