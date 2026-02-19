package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/msenjem/aegis/api/email"
)

type contactRequest struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	Organization string `json:"organization"`
	Role         string `json:"role"`
	Message      string `json:"message"`
}

func (s *Server) ContactForm(w http.ResponseWriter, r *http.Request) {
	var req contactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Email == "" {
		s.writeError(w, http.StatusBadRequest, "email is required")
		return
	}
	if req.Message == "" {
		s.writeError(w, http.StatusBadRequest, "message is required")
		return
	}
	if len(req.Message) > 5000 {
		s.writeError(w, http.StatusBadRequest, "message too long (max 5000 characters)")
		return
	}

	subject, body := email.ContactForm(req.Name, req.Email, req.Organization, req.Role, req.Message)

	to := s.cfg.ContactEmail
	if to == "" {
		to = "contact@aegisimaging.ai"
	}

	if err := s.mailer.Send(r.Context(), to, subject, body); err != nil {
		log.Printf("contact form email error: %v", err)
		s.writeError(w, http.StatusInternalServerError, "failed to send message")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
}
