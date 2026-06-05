package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/kaitoyama/crispy-waffle/internal/domain"
)

// UpsertActor inserts or replaces an actor (used by seeding).
func (s *Store) UpsertActor(ctx context.Context, a domain.Actor) error {
	roles, _ := json.Marshal(a.Roles)
	caps, _ := json.Marshal(a.Capabilities)
	var deleg any
	if a.DelegatedFrom != nil {
		deleg = string(*a.DelegatedFrom)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO actors(id,display_name,kind,identity_ref,roles,delegated_from,capabilities)
		VALUES(?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
		  display_name=excluded.display_name, kind=excluded.kind,
		  identity_ref=excluded.identity_ref, roles=excluded.roles,
		  delegated_from=excluded.delegated_from, capabilities=excluded.capabilities`,
		a.ID, a.DisplayName, a.Kind, a.IdentityRef, string(roles), deleg, string(caps))
	return err
}

// ResolveActor implements workflow.ActorResolver.
func (s *Store) ResolveActor(ctx context.Context, id domain.ActorID) (domain.Actor, error) {
	return s.GetActor(ctx, id)
}

func (s *Store) GetActor(ctx context.Context, id domain.ActorID) (domain.Actor, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id,display_name,kind,identity_ref,roles,delegated_from,capabilities FROM actors WHERE id=?`, id)
	return scanActor(row)
}

func (s *Store) ListActors(ctx context.Context) ([]domain.Actor, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id,display_name,kind,identity_ref,roles,delegated_from,capabilities FROM actors ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Actor
	for rows.Next() {
		a, err := scanActor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanActor(row scanner) (domain.Actor, error) {
	var a domain.Actor
	var roles, caps string
	var deleg sql.NullString
	if err := row.Scan(&a.ID, &a.DisplayName, &a.Kind, &a.IdentityRef, &roles, &deleg, &caps); err != nil {
		return domain.Actor{}, err
	}
	_ = json.Unmarshal([]byte(roles), &a.Roles)
	_ = json.Unmarshal([]byte(caps), &a.Capabilities)
	if deleg.Valid && deleg.String != "" {
		d := domain.ActorID(deleg.String)
		a.DelegatedFrom = &d
	}
	return a, nil
}
