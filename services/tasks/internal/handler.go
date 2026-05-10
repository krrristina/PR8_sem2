package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/krrristina/PR8_sem2/proto"
	"github.com/krrristina/PR8_sem2/shared/middleware"
	"github.com/krrristina/PR8_sem2/shared/sanitize"
)

type Handler struct {
	AuthClient pb.AuthServiceClient
	Log        *zap.Logger
	Repo       *TaskRepository
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

func (h *Handler) verifyToken(ctx context.Context, token, reqID string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-request-id", reqID))

	h.Log.Info("calling grpc verify",
		zap.String("request_id", reqID),
		zap.String("component", "auth_client"),
	)

	resp, err := h.AuthClient.Verify(ctx, &pb.VerifyRequest{Token: token})
	if err != nil {
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.Unauthenticated:
			h.Log.Warn("grpc verify: unauthenticated",
				zap.String("request_id", reqID),
				zap.String("component", "auth_client"),
			)
			return "", fmt.Errorf("unauthorized")
		case codes.DeadlineExceeded:
			h.Log.Error("grpc verify: deadline exceeded",
				zap.String("request_id", reqID),
				zap.String("component", "auth_client"),
			)
			return "", fmt.Errorf("auth unavailable")
		default:
			h.Log.Error("grpc verify: unexpected error",
				zap.String("request_id", reqID),
				zap.String("component", "auth_client"),
				zap.Error(err),
			)
			return "", fmt.Errorf("auth unavailable")
		}
	}

	if !resp.Valid {
		return "", fmt.Errorf("unauthorized")
	}
	return resp.Subject, nil
}

// GetTasks — возвращает все задачи из БД
func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	token := extractToken(r)
	if token == "" {
		h.Log.Warn("missing token",
			zap.String("request_id", reqID),
			zap.String("component", "handler"),
		)
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	_, err := h.verifyToken(r.Context(), token, reqID)
	if err != nil {
		if err.Error() == "unauthorized" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		} else {
			http.Error(w, "auth service unavailable", http.StatusServiceUnavailable)
		}
		return
	}

	tasks, err := h.Repo.GetAll(r.Context())
	if err != nil {
		h.Log.Error("failed to get tasks",
			zap.String("request_id", reqID),
			zap.Error(err),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// SearchTasks — безопасный поиск задач по названию
func (h *Handler) SearchTasks(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	token := extractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	_, err := h.verifyToken(r.Context(), token, reqID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	title := r.URL.Query().Get("title")

	tasks, err := h.Repo.SearchSafe(r.Context(), title)
	if err != nil {
		h.Log.Error("failed to search tasks",
			zap.String("request_id", reqID),
			zap.Error(err),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

// CreateTask — создание новой задачи
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	token := extractToken(r)
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	_, err := h.verifyToken(r.Context(), token, reqID)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// Санитизация — убираем потенциально опасные HTML символы
	body.Description = sanitize.Description(body.Description)

	task, err := h.Repo.Create(r.Context(), body.Title, body.Description)
	if err != nil {
		h.Log.Error("failed to create task",
			zap.String("request_id", reqID),
			zap.Error(err),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}
