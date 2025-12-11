package service

type ServiceMetrics struct {
	metrics *MetricsCollector
}

type MetricsCollector struct {
	updateTasksCount            func(count int)
	updateTasksCountByStatus    func(status string, count int)
	updateTasksCountByPriority  func(priority string, count int)
	incrementCacheHits          func()
	incrementCacheMisses        func()
	incrementDatabaseErrors     func()
	incrementCacheErrors        func()
	incrementValidationErrors   func()
}

func NewMetricsCollector(
	updateTasksCount func(int),
	updateTasksCountByStatus func(string, int),
	updateTasksCountByPriority func(string, int),
	incrementCacheHits func(),
	incrementCacheMisses func(),
	incrementDatabaseErrors func(),
	incrementCacheErrors func(),
	incrementValidationErrors func(),
) *MetricsCollector {
	return &MetricsCollector{
		updateTasksCount:           updateTasksCount,
		updateTasksCountByStatus:   updateTasksCountByStatus,
		updateTasksCountByPriority: updateTasksCountByPriority,
		incrementCacheHits:         incrementCacheHits,
		incrementCacheMisses:       incrementCacheMisses,
		incrementDatabaseErrors:    incrementDatabaseErrors,
		incrementCacheErrors:       incrementCacheErrors,
		incrementValidationErrors:  incrementValidationErrors,
	}
}

func (m *MetricsCollector) UpdateTasksCount(count int) {
	if m.updateTasksCount != nil {
		m.updateTasksCount(count)
	}
}

func (m *MetricsCollector) UpdateTasksCountByStatus(status string, count int) {
	if m.updateTasksCountByStatus != nil {
		m.updateTasksCountByStatus(status, count)
	}
}

func (m *MetricsCollector) UpdateTasksCountByPriority(priority string, count int) {
	if m.updateTasksCountByPriority != nil {
		m.updateTasksCountByPriority(priority, count)
	}
}

func (m *MetricsCollector) IncrementCacheHits() {
	if m.incrementCacheHits != nil {
		m.incrementCacheHits()
	}
}

func (m *MetricsCollector) IncrementCacheMisses() {
	if m.incrementCacheMisses != nil {
		m.incrementCacheMisses()
	}
}

func (m *MetricsCollector) IncrementDatabaseErrors() {
	if m.incrementDatabaseErrors != nil {
		m.incrementDatabaseErrors()
	}
}

func (m *MetricsCollector) IncrementCacheErrors() {
	if m.incrementCacheErrors != nil {
		m.incrementCacheErrors()
	}
}

func (m *MetricsCollector) IncrementValidationErrors() {
	if m.incrementValidationErrors != nil {
		m.incrementValidationErrors()
	}
}