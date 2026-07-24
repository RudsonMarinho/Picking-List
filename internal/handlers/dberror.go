package handlers

import (
	"errors"

	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
)

// mapDBError traduz erros de banco em respostas HTTP consistentes.
// DELETE/UPDATE que violam ON DELETE RESTRICT (tipo/setor/filial em uso)
// devolvem 409, conforme o Contrato da API.
func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "registro não encontrado")
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503": // foreign_key_violation — ON DELETE RESTRICT (em uso) ou referência inválida na criação
			return fiber.NewError(fiber.StatusConflict, "violação de integridade referencial: registro em uso ou referência inválida")
		case "23505": // unique_violation
			return fiber.NewError(fiber.StatusConflict, "registro duplicado")
		case "23514", "23502": // check_violation, not_null_violation
			return fiber.NewError(fiber.StatusBadRequest, "dados inválidos: "+pgErr.Message)
		case "22001": // string_data_right_truncation — valor maior que o tamanho da coluna
			return fiber.NewError(fiber.StatusBadRequest, "dados inválidos: valor excede o tamanho máximo permitido")
		}
	}

	return fiber.NewError(fiber.StatusInternalServerError, "falha ao acessar o banco")
}
