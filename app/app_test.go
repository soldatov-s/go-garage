// nolint:dupl // may be duplications in tests
package app_test

import (
	"context"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/app"
	"github.com/soldatov-s/go-garage/base"
	"github.com/soldatov-s/go-garage/log"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func TestAdd(t *testing.T) {
	testCases := []struct {
		testName      string
		meta          *app.Meta
		expectedError error
		action        func(ctx context.Context, ctrl *gomock.Controller, manager *app.Manager) error
	}{
		{
			testName: "success",
			meta:     nil,
			action: func(ctx context.Context, ctrl *gomock.Controller, manager *app.Manager) error {
				enity := NewMockEnityGateway(ctrl)
				enity.EXPECT().GetFullName().Return("testEnity").AnyTimes()

				if err := manager.Add(ctx, enity); err != nil {
					return errors.Wrap(err, "add enity")
				}

				return nil
			},
			expectedError: nil,
		},
		{
			testName: "conflict",
			meta:     nil,
			action: func(ctx context.Context, ctrl *gomock.Controller, manager *app.Manager) error {
				enity1 := NewMockEnityGateway(ctrl)
				enity1.EXPECT().GetFullName().Return("testEnity").AnyTimes()

				if err := manager.Add(ctx, enity1); err != nil {
					return errors.Wrap(err, "add enity")
				}

				enity2 := NewMockEnityGateway(ctrl)
				enity2.EXPECT().GetFullName().Return("testEnity").AnyTimes()

				if err := manager.Add(ctx, enity2); err != nil {
					return errors.Wrap(err, "add enity")
				}

				return nil
			},
			expectedError: base.ErrConflictName,
		},
		{
			testName: "differebt name",
			meta:     nil,
			action: func(ctx context.Context, ctrl *gomock.Controller, manager *app.Manager) error {
				enity1 := NewMockEnityGateway(ctrl)
				enity1.EXPECT().GetFullName().Return("testEnity1").AnyTimes()
				if err := manager.Add(ctx, enity1); err != nil {
					return errors.Wrap(err, "add enity")
				}

				enity2 := NewMockEnityGateway(ctrl)
				enity2.EXPECT().GetFullName().Return("testEnity2").AnyTimes()

				if err := manager.Add(ctx, enity2); err != nil {
					return errors.Wrap(err, "add enity")
				}

				return nil
			},
			expectedError: nil,
		},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run(tt.testName, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()
			runner, ctx := errgroup.WithContext(ctx)

			cfg := log.DefaultConfig()
			cfg.Level = "DEBUG"
			logger, err := log.NewLogger(ctx, cfg)
			assert.Nil(t, err)
			ctx = logger.Zerolog().WithContext(ctx)

			manager := app.NewManager(&app.ManagerDeps{
				Meta:       tt.meta,
				Logger:     logger,
				ErrorGroup: runner,
			})

			err = tt.action(ctx, ctrl, manager)
			assert.ErrorIs(t, err, tt.expectedError)
		})
	}
}
