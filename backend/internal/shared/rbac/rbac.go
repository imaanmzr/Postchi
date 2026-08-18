package rbac

import (
	"context"
	"errors"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

var roleRank = map[Role]int{
	RoleViewer: 1,
	RoleEditor: 2,
	RoleOwner:  3,
}

var ErrForbidden = errors.New("forbidden")
var ErrNotMember = errors.New("not a workspace member")

func HasMinRole(actual, required Role) bool {
	return roleRank[actual] >= roleRank[required]
}

func CanRead(role Role) bool   { return HasMinRole(role, RoleViewer) }
func CanEdit(role Role) bool   { return HasMinRole(role, RoleEditor) }
func CanDelete(role Role) bool { return role == RoleOwner }
func CanManageMembers(role Role) bool { return role == RoleOwner }

type WorkspaceRoleKey struct{}

func WithWorkspaceRole(ctx context.Context, role Role) context.Context {
	return context.WithValue(ctx, WorkspaceRoleKey{}, role)
}

func WorkspaceRoleFromContext(ctx context.Context) (Role, bool) {
	role, ok := ctx.Value(WorkspaceRoleKey{}).(Role)
	return role, ok
}
