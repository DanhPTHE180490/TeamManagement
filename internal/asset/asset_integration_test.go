package asset_test

import (
	"context"
	"testing"

	"team-management/internal/asset"
	"team-management/internal/auth"
	testutil "team-management/internal/utils"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestAssetCreateShareAndAccess(t *testing.T) {
	db := testutil.InitAndResetDB(t)
	defer db.Close()

	miniRedis := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})
	defer redisClient.Close()

	authRepo := auth.NewAuthRepository(db)
	authSvc := auth.NewAuthService(authRepo)

	owner, err := authSvc.Register(context.Background(), "asset-owner", "asset-owner@example.com", "password123", "member")
	if err != nil {
		t.Fatalf("failed to register owner: %v", err)
	}
	other, err := authSvc.Register(context.Background(), "asset-other", "asset-other@example.com", "password123", "member")
	if err != nil {
		t.Fatalf("failed to register other user: %v", err)
	}

	assetRepo := asset.NewAssetRepository(db, redisClient)
	assetSvc := asset.NewAssetService(assetRepo)

	folder, err := assetSvc.CreateFolder(context.Background(), int64(owner.ID), "My Folder")
	if err != nil {
		t.Fatalf("failed to create folder: %v", err)
	}

	note, err := assetSvc.CreateNote(context.Background(), int64(owner.ID), "IT Test Note", "some content", &folder.ID)
	if err != nil {
		t.Fatalf("failed to create note: %v", err)
	}

	if err := assetSvc.ShareNote(context.Background(), int64(owner.ID), note.ID, int64(other.ID), "read"); err != nil {
		t.Fatalf("failed to share note: %v", err)
	}

	// other user should be able to see shared notes
	shared, err := assetSvc.GetSharedNotes(context.Background(), int64(other.ID))
	if err != nil {
		t.Fatalf("failed to get shared notes: %v", err)
	}
	if len(shared) == 0 {
		t.Fatalf("expected at least one shared note for user %d", other.ID)
	}
}
