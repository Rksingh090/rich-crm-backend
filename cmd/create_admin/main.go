package main

import (
	"context"
	"go-crm/internal/common/models"
	"go-crm/internal/config"
	"go-crm/internal/core/user"
	"go-crm/internal/database"
	"go-crm/internal/logger"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func CreateAdmin(
	lc fx.Lifecycle,
	userRepo user.UserRepository,
	logger *zap.Logger,
	shutdowner fx.Shutdowner,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				defer func() {
					if err := shutdowner.Shutdown(); err != nil {
						logger.Error("Failed to shutdown", zap.Error(err))
					}
				}()

				logger.Info("👤 Creating Global Super Admin...")

				password := "Rksingh@090" // Change this!
				email := "admin@platform.com"

				// Check if exists
				existing, err := userRepo.FindByEmailGlobal(ctx, email)
				if err == nil {
					logger.Info("Super Admin already exists", zap.String("id", existing.ID.Hex()))
					return
				}

				// Create User with Zero TenantID
				newUser := models.User{
					ID:       primitive.NewObjectID(),
					Password: password,
					Email:    email,
					Status:   "active",
					// Roles removed in favor of AppRoles
					IsPlatformAdmin: true,
					TenantID:        primitive.NilObjectID, // explicit nil/zero
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				}

				// Hack: Repository might enforce TenantID from context.
				// We need to bypass or ensure repo handles nil tenantID for global users.
				// Let's check repository.go...
				// repo.Create enforces tenantID from context:
				//   tenantID, ok := ctx.Value(models.TenantIDKey).(string)
				//   if !ok || tenantID == "" { return fmt.Errorf("tenant context missing") }
				// So we MUST pass a "zero" tenant ID in context or update repo to allow "system" context.

				// Let's pass a special "global" context
				globalCtx := context.WithValue(ctx, models.TenantIDKey, "000000000000000000000000")

				// But repo tries to ParseObjectID(tenantID).
				// "000000000000000000000000" is a valid 24 char hex 0 object ID.
				// So it should parse to primitive.NilObjectID (or zero value).

				if err := userRepo.Create(globalCtx, &newUser); err != nil {
					logger.Fatal("Failed to create super admin", zap.Error(err))
				}

				logger.Info("✅ Super Admin Created Successfully!",
					zap.String("email", email),
					zap.String("password", password),
				)
			}()
			return nil
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(
			config.LoadConfig,
			logger.NewLogger,
			database.NewDatabase,
			user.NewUserRepository,
		),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		fx.Invoke(CreateAdmin),
	)

	if err := app.Start(context.Background()); err != nil {
		log.Fatal(err)
	}

	<-app.Done()
}
