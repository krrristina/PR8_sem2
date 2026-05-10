package grpc

import (
	"context"
	"strings"

	pb "github.com/krrristina/PR8_sem2/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthGRPCServer struct {
	pb.UnimplementedAuthServiceServer
	Log *zap.Logger
}

func (s *AuthGRPCServer) Verify(ctx context.Context, req *pb.VerifyRequest) (*pb.VerifyResponse, error) {
	// Достаём request-id из gRPC метаданных
	reqID := ""
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("x-request-id"); len(vals) > 0 {
			reqID = vals[0]
		}
	}

	s.Log.Info("verify called",
		zap.String("request_id", reqID),
		zap.Bool("has_token", req.Token != ""),
	)

	if req.Token == "" {
		s.Log.Warn("empty token",
			zap.String("request_id", reqID),
		)
		return nil, status.Error(codes.Unauthenticated, "invalid token: token is empty")
	}

	if strings.HasPrefix(req.Token, "invalid") {
		s.Log.Warn("invalid token",
			zap.String("request_id", reqID),
		)
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	s.Log.Info("token valid",
		zap.String("request_id", reqID),
	)
	return &pb.VerifyResponse{
		Valid:   true,
		Subject: "user@example.com",
	}, nil
}
