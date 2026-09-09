package repo_test

import (
	"context"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func TestBlogRepo_CreateGetUpdateDelete(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	blog := repo.NewBlogRepo(pool)
	ctx := context.Background()

	post, err := blog.Create(ctx, &models.BlogPost{
		Slug: "hello-world", Title: "Hello World", BodyHTML: "<p>Hi</p>", BodyText: "Hi",
		AuthorName: "Admin", Published: false,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if post.PublishedAt != nil {
		t.Error("expected published_at to be nil for an unpublished post")
	}

	fetched, err := blog.GetBySlug(ctx, "hello-world")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if fetched.Title != "Hello World" {
		t.Errorf("got title %q, want Hello World", fetched.Title)
	}

	if _, err := blog.GetPublishedBySlug(ctx, "hello-world"); err != repo.ErrNotFound {
		t.Errorf("expected unpublished post to be hidden from GetPublishedBySlug, got err=%v", err)
	}

	// Publishing for the first time should set published_at.
	fetched.Published = true
	updated, err := blog.Update(ctx, fetched, false)
	if err != nil {
		t.Fatalf("Update (publish): %v", err)
	}
	if updated.PublishedAt == nil {
		t.Error("expected published_at to be set after publishing")
	}
	firstPublishedAt := *updated.PublishedAt

	if _, err := blog.GetPublishedBySlug(ctx, "hello-world"); err != nil {
		t.Errorf("expected published post to be visible via GetPublishedBySlug, got %v", err)
	}

	// Editing an already-published post again should NOT change published_at.
	updated.Title = "Hello World, Updated"
	updated2, err := blog.Update(ctx, updated, true)
	if err != nil {
		t.Fatalf("Update (re-edit): %v", err)
	}
	if updated2.PublishedAt == nil || !updated2.PublishedAt.Equal(firstPublishedAt) {
		t.Errorf("expected published_at to stay %v, got %v", firstPublishedAt, updated2.PublishedAt)
	}

	if err := blog.Delete(ctx, post.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := blog.GetByID(ctx, post.ID); err != repo.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestBlogRepo_ListPublishedExcludesDrafts(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	blog := repo.NewBlogRepo(pool)
	ctx := context.Background()

	if _, err := blog.Create(ctx, &models.BlogPost{
		Slug: "published-post", Title: "Published", BodyHTML: "<p>A</p>", BodyText: "A", Published: true,
	}); err != nil {
		t.Fatalf("Create published: %v", err)
	}
	if _, err := blog.Create(ctx, &models.BlogPost{
		Slug: "draft-post", Title: "Draft", BodyHTML: "<p>B</p>", BodyText: "B", Published: false,
	}); err != nil {
		t.Fatalf("Create draft: %v", err)
	}

	published, total, err := blog.ListPublished(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListPublished: %v", err)
	}
	if total != 1 || len(published) != 1 || published[0].Slug != "published-post" {
		t.Fatalf("expected only the published post, got total=%d posts=%+v", total, published)
	}

	all, allTotal, err := blog.ListAll(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if allTotal != 2 || len(all) != 2 {
		t.Fatalf("expected both posts in ListAll, got total=%d len=%d", allTotal, len(all))
	}
}

func TestBlogRepo_SlugExists(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	blog := repo.NewBlogRepo(pool)
	ctx := context.Background()

	if _, err := blog.Create(ctx, &models.BlogPost{
		Slug: "taken", Title: "Taken", BodyHTML: "<p>A</p>", BodyText: "A",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	exists, err := blog.SlugExists(ctx, "taken")
	if err != nil || !exists {
		t.Errorf("expected 'taken' to exist, got exists=%v err=%v", exists, err)
	}
	exists, err = blog.SlugExists(ctx, "not-taken")
	if err != nil || exists {
		t.Errorf("expected 'not-taken' to not exist, got exists=%v err=%v", exists, err)
	}
}
