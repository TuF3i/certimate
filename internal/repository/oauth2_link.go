package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/certimate-go/certimate/internal/app"
	"github.com/certimate-go/certimate/internal/domain"
)

type OAuth2LinkRepository struct{}

func NewOAuth2LinkRepository() *OAuth2LinkRepository {
	return &OAuth2LinkRepository{}
}

func (r *OAuth2LinkRepository) GetByProviderAndSubject(ctx context.Context, provider, subjectId string) (*domain.OAuth2Link, error) {
	record, err := app.GetApp().FindFirstRecordByFilter(
		domain.CollectionNameOAuth2Link,
		"provider={:provider} && subjectId={:subjectId}",
		dbx.Params{"provider": provider, "subjectId": subjectId},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrRecordNotFound
		}
		return nil, err
	}
	return r.castRecordToModel(record)
}

func (r *OAuth2LinkRepository) ListBySuperuser(ctx context.Context, superuserId string) ([]*domain.OAuth2Link, error) {
	records, err := app.GetApp().FindRecordsByFilter(
		domain.CollectionNameOAuth2Link,
		"superuserId={:superuserId}",
		"-created",
		0, 0,
		dbx.Params{"superuserId": superuserId},
	)
	if err != nil {
		return nil, err
	}

	links := make([]*domain.OAuth2Link, 0, len(records))
	for _, record := range records {
		link, err := r.castRecordToModel(record)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

func (r *OAuth2LinkRepository) Save(ctx context.Context, link *domain.OAuth2Link) (*domain.OAuth2Link, error) {
	collection, err := app.GetApp().FindCollectionByNameOrId(domain.CollectionNameOAuth2Link)
	if err != nil {
		return link, err
	}

	var record *core.Record
	if link.Id == "" {
		record = core.NewRecord(collection)
	} else {
		record, err = app.GetApp().FindRecordById(collection, link.Id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return link, domain.ErrRecordNotFound
			}
			return link, err
		}
	}

	record.Set("provider", link.Provider)
	record.Set("subjectId", link.SubjectId)
	record.Set("superuserId", link.SuperuserId)
	if link.TargetCollection != "" {
		record.Set("targetCollection", link.TargetCollection)
	} else {
		record.Set("targetCollection", "_superusers")
	}
	if link.UserProfileEmail != "" {
		record.Set("userProfileEmail", link.UserProfileEmail)
	}
	if link.UserProfileName != "" {
		record.Set("userProfileName", link.UserProfileName)
	}
	if link.UserProfileAvatar != "" {
		record.Set("userProfileAvatar", link.UserProfileAvatar)
	}

	if err := app.GetApp().Save(record); err != nil {
		return link, err
	}

	link.Id = record.Id
	link.CreatedAt = record.GetDateTime("created").Time()
	link.UpdatedAt = record.GetDateTime("updated").Time()
	return link, nil
}

func (r *OAuth2LinkRepository) Delete(ctx context.Context, id string) error {
	record, err := app.GetApp().FindRecordById(domain.CollectionNameOAuth2Link, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrRecordNotFound
		}
		return err
	}
	return app.GetApp().Delete(record)
}

func (r *OAuth2LinkRepository) castRecordToModel(record *core.Record) (*domain.OAuth2Link, error) {
	if record == nil {
		return nil, fmt.Errorf("the record is nil")
	}

	return &domain.OAuth2Link{
		Meta: domain.Meta{
			Id:        record.Id,
			CreatedAt: record.GetDateTime("created").Time(),
			UpdatedAt: record.GetDateTime("updated").Time(),
		},
		Provider:          record.GetString("provider"),
		SubjectId:         record.GetString("subjectId"),
		TargetCollection:  record.GetString("targetCollection"),
		SuperuserId:       record.GetString("superuserId"),
		UserProfileEmail:  record.GetString("userProfileEmail"),
		UserProfileName:   record.GetString("userProfileName"),
		UserProfileAvatar: record.GetString("userProfileAvatar"),
	}, nil
}
