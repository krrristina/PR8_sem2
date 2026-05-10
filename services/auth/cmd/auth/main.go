package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	pb "github.com/krrristina/PR8_sem2/proto"
	grpcserver "github.com/krrristina/PR8_sem2/services/auth/internal/grpc"
	"github.com/krrristina/PR8_sem2/shared/logger"
	"github.com/krrristina/PR8_sem2/shared/middleware"
)

func main() {
	log, err := logger.New("auth")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	// gRPC сервер
	port := os.Getenv("AUTH_GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("failed to listen", zap.Error(err))
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &grpcserver.AuthGRPCServer{Log: log})

	log.Info("gRPC Auth server starting", zap.String("port", port))

	// Запускаем gRPC в фоне
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatal("failed to serve grpc", zap.Error(err))
		}
	}()

	// HTTP сервер для логина и выдачи cookies
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", loginHandler(log))

	handler := middleware.SecurityHeaders(mux)

	httpPort := os.Getenv("AUTH_HTTP_PORT")
	if httpPort == "" {
		httpPort = "8081"
	}

	log.Info("HTTP Auth server starting", zap.String("port", httpPort))
	if err := http.ListenAndServe(":"+httpPort, handler); err != nil {
		log.Fatal("failed to serve http", zap.Error(err))
	}
}

// generateToken генерирует случайный токен
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// loginHandler — выдаёт session и csrf_token cookies
func loginHandler(log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}

		// Учебная проверка: фиксированные credentials
		if body.Username != "student" || body.Password != "student" {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		sessionToken := generateToken()
		csrfToken := generateToken()

		// session cookie — HttpOnly (JS не может прочитать)
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true, // защита от XSS
			Secure:   true, // только HTTPS
			SameSite: http.SameSiteLaxMode,
		})

		// csrf_token cookie — НЕ HttpOnly (фронт должен прочитать)
		http.SetCookie(w, &http.Cookie{
			Name:     "csrf_token",
			Value:    csrfToken,
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: false, // фронт читает и отправляет заголовком
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})

		log.Info("login successful",
			zap.String("username", body.Username),
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"csrf_token": csrfToken,
			"message":    "login successful",
		})
	}
}
