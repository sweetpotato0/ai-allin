package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ================================
// API服务器
// ================================

// APIServer HTTP API服务
type APIServer struct {
	platform *ECommerceServicePlatform
	server   *http.Server
}

// API请求/响应模型
type (
	// CustomerInquiryRequest 客户咨询请求
	CustomerInquiryRequest struct {
		CustomerID string `json:"customer_id"`
		Message    string `json:"message"`
		SessionID  string `json:"session_id,omitempty"`
	}

	// CustomerInquiryResponse 客户咨询响应
	CustomerInquiryResponse struct {
		SessionID string  `json:"session_id"`
		Response  string  `json:"response"`
		TicketID  string  `json:"ticket_id"`
		Messages  int     `json:"messages_count"`
		Duration  float64 `json:"duration_ms"`
		Success   bool    `json:"success"`
		Error     string  `json:"error,omitempty"`
	}

	// SessionInfoRequest 会话信息请求
	SessionInfoRequest struct {
		SessionID string `json:"session_id"`
	}

	// SessionInfoResponse 会话信息响应
	SessionInfoResponse struct {
		SessionID      string `json:"session_id"`
		Status         string `json:"status"`
		MessageCount   int    `json:"message_count"`
		CreatedAt      string `json:"created_at"`
		LastAccessedAt string `json:"last_accessed_at"`
	}

	// HealthCheckResponse 健康检查响应
	HealthCheckResponse struct {
		Status            string         `json:"status"`
		Timestamp         string         `json:"timestamp"`
		ActiveSessions    int            `json:"active_sessions"`
		Metrics           map[string]any `json:"metrics"`
		DatabaseConnected bool           `json:"database_connected"`
		RedisConnected    bool           `json:"redis_connected"`
		CacheHitRate      float64        `json:"cache_hit_rate"`
	}

	// ErrorResponse 错误响应
	ErrorResponse struct {
		Error      string `json:"error"`
		Message    string `json:"message"`
		StatusCode int    `json:"status_code"`
		Timestamp  string `json:"timestamp"`
	}
)

// NewAPIServer 创建API服务器
func NewAPIServer(platform *ECommerceServicePlatform, port string) *APIServer {
	api := &APIServer{
		platform: platform,
	}

	// 创建HTTP服务器
	mux := http.NewServeMux()

	// 路由注册
	mux.HandleFunc("/api/v1/inquiry", api.handleInquiry)
	mux.HandleFunc("/api/v1/session", api.handleSession)
	mux.HandleFunc("/api/v1/sessions", api.listSessions)
	mux.HandleFunc("/api/v1/metrics", api.getMetrics)
	mux.HandleFunc("/api/v1/health", api.healthCheck)
	mux.HandleFunc("/api/v1/customers", api.getCustomers)
	mux.HandleFunc("/api/v1/orders", api.getOrders)
	mux.HandleFunc("/api/v1/tickets", api.getTickets)

	api.server = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return api
}

// Start 启动API服务器
func (s *APIServer) Start() error {
	return s.server.ListenAndServe()
}

// Stop 停止API服务器
func (s *APIServer) Stop(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// ================================
// API处理方法
// ================================

// handleInquiry 处理客户咨询
func (s *APIServer) handleInquiry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持POST请求")
		return
	}

	var req CustomerInquiryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("请求格式错误: %v", err))
		return
	}

	// 验证请求
	if req.CustomerID == "" || req.Message == "" {
		s.writeError(w, http.StatusBadRequest, "customer_id和message不能为空")
		return
	}

	// 处理咨询
	startTime := time.Now()
	response, err := s.platform.HandleCustomerInquiry(req.CustomerID, req.Message)
	duration := time.Since(startTime).Milliseconds()

	// 获取工单（这里简化了）
	ticketID := fmt.Sprintf("TKT_%d", time.Now().Unix())

	resp := CustomerInquiryResponse{
		SessionID: req.SessionID,
		Response:  response,
		TicketID:  ticketID,
		Duration:  float64(duration),
		Success:   err == nil,
	}

	if err != nil {
		resp.Error = err.Error()
	}

	s.writeJSON(w, http.StatusOK, resp)
}

// handleSession 处理会话操作
func (s *APIServer) handleSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		s.writeError(w, http.StatusBadRequest, "session_id参数不能为空")
		return
	}

	if r.Method == http.MethodGet {
		// 获取会话信息
		resp := SessionInfoResponse{
			SessionID: sessionID,
			Status:    "active",
			CreatedAt: time.Now().AddDate(0, 0, -1).Format(time.RFC3339),
		}
		s.writeJSON(w, http.StatusOK, resp)
	} else if r.Method == http.MethodDelete {
		// 删除会话
		s.platform.sessionManager.Delete(context.Background(), sessionID)
		s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	} else {
		s.writeError(w, http.StatusMethodNotAllowed, "不支持的HTTP方法")
	}
}

// listSessions 列表所有会话
func (s *APIServer) listSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持GET请求")
		return
	}

	sessionIDs, err := s.platform.sessionManager.List(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("获取会话列表失败: %v", err))
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"sessions": sessionIDs,
		"count":    len(sessionIDs),
	})
}

// getMetrics 获取平台指标
func (s *APIServer) getMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持GET请求")
		return
	}

	s.platform.metrics.mu.RLock()
	defer s.platform.metrics.mu.RUnlock()

	activeSessions, err := s.platform.sessionManager.Count(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("获取会话数量失败: %v", err))
		return
	}

	metrics := map[string]any{
		"total_requests":       s.platform.metrics.TotalRequests,
		"successful_requests":  s.platform.metrics.SuccessfulRequests,
		"failed_requests":      s.platform.metrics.FailedRequests,
		"avg_response_time_ms": s.platform.metrics.AverageResponseTime.Milliseconds(),
		"active_sessions":      activeSessions,
		"success_rate": float64(s.platform.metrics.SuccessfulRequests) /
			float64(s.platform.metrics.TotalRequests) * 100,
	}

	s.writeJSON(w, http.StatusOK, metrics)
}

// healthCheck 健康检查
func (s *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持GET请求")
		return
	}

	s.platform.metrics.mu.RLock()
	defer s.platform.metrics.mu.RUnlock()

	activeSessions, err := s.platform.sessionManager.Count(r.Context())
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("获取会话数量失败: %v", err))
		return
	}

	resp := HealthCheckResponse{
		Status:            "healthy",
		Timestamp:         time.Now().Format(time.RFC3339),
		ActiveSessions:    activeSessions,
		DatabaseConnected: true, // 实际应该检查真实数据库
		RedisConnected:    true, // 实际应该检查真实Redis
		CacheHitRate:      0.95,
		Metrics: map[string]any{
			"total_requests":      s.platform.metrics.TotalRequests,
			"successful_requests": s.platform.metrics.SuccessfulRequests,
			"failed_requests":     s.platform.metrics.FailedRequests,
		},
	}

	statusCode := http.StatusOK
	if s.platform.metrics.FailedRequests > s.platform.metrics.SuccessfulRequests {
		resp.Status = "degraded"
		statusCode = http.StatusServiceUnavailable
	}

	s.writeJSON(w, statusCode, resp)
}

// getCustomers 获取客户列表
func (s *APIServer) getCustomers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持GET请求")
		return
	}

	s.platform.customersMutex.RLock()
	defer s.platform.customersMutex.RUnlock()

	customers := make([]map[string]any, 0)
	for _, c := range s.platform.customers {
		customers = append(customers, map[string]any{
			"id":            c.ID,
			"name":          c.Name,
			"vip_level":     c.VIPLevel,
			"total_spent":   c.TotalSpent,
			"order_count":   c.OrderCount,
			"registered_at": c.RegisteredAt,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"customers": customers,
		"count":     len(customers),
	})
}

// getOrders 获取订单列表
func (s *APIServer) getOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持GET请求")
		return
	}

	s.platform.ordersMutex.RLock()
	defer s.platform.ordersMutex.RUnlock()

	orders := make([]map[string]any, 0)
	for _, o := range s.platform.orders {
		orders = append(orders, map[string]any{
			"order_id":     o.OrderID,
			"customer_id":  o.CustomerID,
			"total_amount": o.TotalAmount,
			"status":       o.Status,
			"created_at":   o.CreatedAt,
			"tracking_url": o.TrackingURL,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"orders": orders,
		"count":  len(orders),
	})
}

// getTickets 获取工单列表
func (s *APIServer) getTickets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持GET请求")
		return
	}

	s.platform.ticketsMutex.RLock()
	defer s.platform.ticketsMutex.RUnlock()

	tickets := make([]map[string]any, 0)
	for _, t := range s.platform.tickets {
		tickets = append(tickets, map[string]any{
			"ticket_id":   t.TicketID,
			"customer_id": t.CustomerID,
			"subject":     t.Subject,
			"priority":    t.Priority,
			"status":      t.Status,
			"created_at":  t.CreatedAt,
			"assigned_to": t.AssignedTo,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"tickets": tickets,
		"count":   len(tickets),
	})
}

// ================================
// 辅助方法
// ================================

// writeJSON 写入JSON响应
func (s *APIServer) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Powered-By", "ai-allin-ecommerce")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// writeError 写入错误响应
func (s *APIServer) writeError(w http.ResponseWriter, statusCode int, message string) {
	resp := ErrorResponse{
		Error:      http.StatusText(statusCode),
		Message:    message,
		StatusCode: statusCode,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}
