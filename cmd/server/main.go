package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	v1 "github.com/led0nk/ark-clusterinfo/api/v1"
	"github.com/led0nk/ark-clusterinfo/internal"
	blist "github.com/led0nk/ark-clusterinfo/internal/blacklist"
	"github.com/led0nk/ark-clusterinfo/internal/events"
	"github.com/led0nk/ark-clusterinfo/internal/jsondb"
	"github.com/led0nk/ark-clusterinfo/internal/notifier"
	"github.com/led0nk/ark-clusterinfo/internal/notifier/services/discord"
	"github.com/led0nk/ark-clusterinfo/internal/overseer"
	"github.com/led0nk/ark-clusterinfo/observer"
	"github.com/led0nk/ark-clusterinfo/pkg/config"
)

func main() {

	var (
		addr   = flag.String("addr", "localhost:8080", "server port")
		db     = flag.String("db", "testdata", "path to the database")
		blpath = flag.String("blacklist", "testdata", "path to the blacklist")
		//grpcaddr    = flag.String("grpcaddr", "", "grpc address, e.g. localhost:4317")
		domain      = flag.String("domain", "127.0.0.1", "given domain for cookies/mail")
		logLevelStr = flag.String("loglevel", "INFO", "define the level for logs")
		sStore      internal.ServerStore
		obs         internal.Observer
		ovs         internal.Overseer
		blacklist   internal.Blacklist
		messaging   internal.Notification
		logLevel    slog.Level
		notSvcName  string
	)
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := logLevel.UnmarshalText([]byte(*logLevelStr))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	if err != nil {
		logger.ErrorContext(ctx, "error parsing loglevel", "loglevel", *logLevelStr, "error", err)
		os.Exit(1)
	}
	slog.SetDefault(logger)

	logger.Info("server address", "addr", *addr)

	cfg, err := config.NewConfiguration("testdata/config.yaml")
	if err != nil {
		logger.Error("failed to create new config", "error", err)
	}
	nService, err := cfg.GetSection("notification-service")
	if err != nil {
		logger.Error("failed to get section from config", "error", err)
	}
	for key, value := range nService {
		switch key {
		case "discord":
			discordConfig, ok := value.(map[interface{}]interface{})
			if !ok {
				logger.Error("discord section has wrong type", "error", "discord")
			}

			token, ok := discordConfig["token"].(string)
			if !ok {
				logger.Error("token was not found or has wrong type", "error", "discord")
			}

			channelID, ok := discordConfig["channelID"].(string)
			if !ok {
				logger.Error("channelID was not found or has wrong type", "error", "discord")
			}

			messaging, err = discord.NewDiscordNotifier(ctx, token, channelID)
			if err != nil {
				logger.ErrorContext(ctx, "failed to create notification service", "error", err)
				os.Exit(1)
			}
			notSvcName = "discord"
		default:
			notSvcName = "none"
			logger.Info("no expected notification service found", "config", notSvcName)
		}
	}

	sStore, err = jsondb.NewServerStorage(*db + "/cluster.json")
	if err != nil {
		logger.ErrorContext(ctx, "failed to create new cluster", "error", err)
		os.Exit(1)
	}

	em := events.NewEventManager()

	notify := notifier.NewNotifier(sStore, em)
	sStore = notify

	obs, err = observer.NewObserver(sStore)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create endpoint storage", "error", err)
		os.Exit(1)
	}

	blacklist, err = blist.NewBlacklist(*blpath + "/blacklist.json")
	if err != nil {
		logger.ErrorContext(ctx, "failed to create blacklist", "error", err)
		os.Exit(1)
	}

	ovs, err = overseer.NewOverseer(ctx, sStore, blacklist, em)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create overseer", "error", err)
		os.Exit(1)
	}

	go em.StartListening(ctx, messaging, notSvcName)
	go em.StartListening(ctx, obs, "observer")
	go em.StartListening(ctx, ovs, "overseer")
	go obs.SpawnScraper(ctx)
	go ovs.SpawnScanner(ctx)

	server := v1.NewServer(*addr, *domain, logger, sStore, blacklist)
	server.ServeHTTP()
}

// NOTE: just to initialize first Targets
//func initTargets(ctx context.Context, sStore internal.ServerStore, logger *slog.Logger) error {
//	sStore.Create(ctx, &model.Server{
//		Name: "Ragnarok",
//		Addr: "51.195.60.114:27019",
//	})
//
//	sStore.Create(ctx, &model.Server{
//		Name: "LostIsland",
//		Addr: "51.195.60.114:27020",
//	})
//
//	sStore.Create(ctx, &model.Server{
//		Name: "Aberration",
//		Addr: "51.195.60.114:27018",
//	})
//
//	sStore.Create(ctx, &model.Server{
//		Name: "TheIsland",
//		Addr: "51.195.60.114:27016",
//	})
//	return nil
//}
//
//func initBlacklist(ctx context.Context, blacklist internal.Blacklist, logger *slog.Logger) error {
//	_, err := blacklist.Create(ctx, &model.Players{
//		Name: "Fadem",
//	})
//	if err != nil {
//		logger.ErrorContext(ctx, "failed to create blacklist entry", "error", err)
//		return err
//	}
//	_, err = blacklist.Create(ctx, &model.Players{
//		Name: "FisherSpider",
//	})
//	if err != nil {
//		logger.ErrorContext(ctx, "failed to create blacklist entry", "error", err)
//		return err
//	}
//	_, err = blacklist.Create(ctx, &model.Players{
//		Name: "Hermes Headquart...",
//	})
//	if err != nil {
//		logger.ErrorContext(ctx, "failed to create blacklist entry", "error", err)
//		return err
//	}
//	return nil
//}
