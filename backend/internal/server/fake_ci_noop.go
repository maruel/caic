// No-op fake CI stub for production builds.
//go:build !e2e

package server

import "github.com/caic-xyz/caic/backend/internal/task"

func (s *Server) maybeFakeCI(_ *task.Task) {}
