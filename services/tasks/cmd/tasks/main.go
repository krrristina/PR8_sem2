package main

import (
	"context"
	"net/http"
	"os"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	pb "github.com/krrristina/PR8_sem2/proto"
	"github.com/krrristina/PR8_sem2/services/tasks/internal"
	"github.com/krrristina/PR8_sem2/shared/logger"
	"github.com/krrristina/PR8_sem2/shared/middleware"
)

func main() {
	log, err := logger.New("tasks")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "postgres://tasks:tasks@localhost:5432/tasks?sslmode=disable"
	}

	repo, err := internal.NewTaskRepository(dsn)
	if err != nil {
		log.Fatal("failed to connect to db", zap.Error(err))
	}
	log.Info("connected to database")

	if err := repo.Migrate(context.Background()); err != nil {
		log.Fatal("failed to migrate", zap.Error(err))
	}
	log.Info("migration done")

	authAddr := os.Getenv("AUTH_GRPC_ADDR")
	if authAddr == "" {
		authAddr = "localhost:50051"
	}

	conn, err := grpc.Dial(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("could not connect to auth", zap.Error(err))
	}
	defer conn.Close()

	h := &internal.Handler{
		AuthClient: pb.NewAuthServiceClient(conn),
		Log:        log,
		Repo:       repo,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreateTask(w, r)
		} else {
			h.GetTasks(w, r)
		}
	})
	mux.HandleFunc("/tasks/search", h.SearchTasks)
	mux.Handle("/metrics", promhttp.Handler())

	// Порядок middleware: SecurityHeaders → RequestID → AccessLog → CSRF → Metrics → mux
	handler := middleware.SecurityHeaders(
		middleware.RequestID(
			middleware.AccessLog(log)(
				middleware.CSRF(
					middleware.Metrics("/tasks")(mux),
				),
			),
		),
	)

	tasksPort := os.Getenv("TASKS_PORT")
	if tasksPort == "" {
		tasksPort = "8082"
	}

	log.Info("HTTP tasks server starting", zap.String("port", tasksPort))
	if err := http.ListenAndServe(":"+tasksPort, handler); err != nil {
		log.Fatal("server failed", zap.Error(err))
	}
}
