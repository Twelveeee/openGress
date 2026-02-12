package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Twelveeee/golib/logger"
	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/google/uuid"
)

type cliOptions struct {
	apiBase string
	token   string
}

func main() {
	cfgPath := flag.String("config", "config/config.yml", "config path")
	apiBase := flag.String("api", "", "admin api base url, e.g. http://localhost:8080")
	adminToken := flag.String("token", "", "admin token (defaults to config admin.token)")
	flag.Parse()

	cfg, err := config.LoadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logg, closeFunc, err := logger.NewLogger(ctx, &cfg.LogConfig)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := closeFunc(); err != nil {
			log.Printf("logger shutdown error: %v", err)
		}
	}()
	if logg != nil {
		slog.SetDefault(logg)
	}

	opts := cliOptions{
		apiBase: strings.TrimRight(*apiBase, "/"),
		token:   strings.TrimSpace(*adminToken),
	}

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "status":
		runStatus(args[1:], cfg, opts)
	case "dump":
		runDump(args[1:], cfg, opts)
	case "load":
		runLoad(args[1:], cfg, opts)
	case "announce":
		runAnnounce(args[1:], cfg, opts)
	case "kick":
		runKick(args[1:], cfg, opts)
	case "ban":
		runBan(args[1:], cfg, opts)
	case "unban":
		runUnban(args[1:], cfg, opts)
	case "player":
		runPlayer(args[1:], cfg, opts)
	case "portal":
		runPortal(args[1:], cfg, opts)
	case "link":
		runLink(args[1:], cfg, opts)
	case "field":
		runField(args[1:], cfg, opts)
	case "logs":
		runLogs(args[1:], cfg, opts)
	case "gc-logs":
		runGCLogs(args[1:], cfg, opts)
	case "recalc-stats":
		runRecalcStats(args[1:], cfg, opts)
	case "reload-config":
		runReloadConfig(args[1:], cfg, opts)
	case "rotate-logs":
		runRotateLogs(args[1:], cfg, opts)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	log.Println("Usage: cli -config=config/config.yml <command> [options]")
	log.Println("  Optional: -api http://localhost:8080 -token <admin-token>")
	log.Println("Commands: status|dump|load|announce|kick|ban|unban|player|portal|link|field|logs|gc-logs|recalc-stats|reload-config|rotate-logs")
}

func runStatus(args []string, cfg *config.Config, opts cliOptions) {
	if client := adminClientFrom(cfg, opts); client != nil {
		var data map[string]interface{}
		if err := client.doJSON(http.MethodGet, "/api/v1/admin/status", nil, &data); err != nil {
			log.Fatalf("status failed: %v", err)
		}
		slog.Info("status", "data", data)
		return
	}
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	_ = fs.Parse(args)

	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}

	logCount := 0
	if gameState.Logs != nil {
		logCount = len(gameState.Logs.List())
	}
	banned := 0
	if gameState.Admin != nil {
		banned = len(gameState.Admin.Banned)
	}

	slog.Info("status",
		"players", len(gameState.Players.List()),
		"portals", len(gameState.Portals.List()),
		"links", len(gameState.Links.List()),
		"fields", len(gameState.Fields.List()),
		"logs", logCount,
		"banned", banned,
	)
}

func runDump(args []string, cfg *config.Config, opts cliOptions) {
	if adminClientFrom(cfg, opts) != nil {
		log.Fatal("dump is offline-only; use -state/-out without -api")
	}
	fs := flag.NewFlagSet("dump", flag.ExitOnError)
	statePath := fs.String("state", "", "input snapshot path (optional)")
	outPath := fs.String("out", "", "output snapshot path")
	_ = fs.Parse(args)

	if *outPath == "" {
		log.Fatal("dump requires -out <path>")
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	if err := saveState(*outPath, gameState); err != nil {
		log.Fatalf("dump failed: %v", err)
	}
	slog.Info("dump completed", "path", *outPath)
}

func runLoad(args []string, cfg *config.Config, opts cliOptions) {
	if adminClientFrom(cfg, opts) != nil {
		log.Fatal("load is offline-only; use -state without -api")
	}
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	inPath := fs.String("in", "", "snapshot path")
	_ = fs.Parse(args)
	if *inPath == "" {
		log.Fatal("load requires -in <path>")
	}
	gameState, err := loadState(*inPath, cfg)
	if err != nil {
		log.Fatalf("load failed: %v", err)
	}
	slog.Info("load completed",
		"path", *inPath,
		"players", len(gameState.Players.List()),
		"portals", len(gameState.Portals.List()),
		"links", len(gameState.Links.List()),
		"fields", len(gameState.Fields.List()),
	)
}

func runAnnounce(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("announce", flag.ExitOnError)
	message := fs.String("message", "", "announcement message")
	statePath := fs.String("state", "", "snapshot path (optional)")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	_ = fs.Parse(args)
	if *message == "" {
		log.Fatal("announce requires -message")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/announce", map[string]interface{}{
			"message": *message,
		}, nil); err != nil {
			log.Fatalf("announce failed: %v", err)
		}
		slog.Info("announce queued", "message", *message)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	appendLog(gameState, state.LogEntry{
		ID:      uuid.New().String(),
		Type:    "ANNOUNCE",
		Message: *message,
	})
	if *statePath != "" || *outPath != "" {
		savePath := pickOutPath(*statePath, *outPath)
		if err := saveState(savePath, gameState); err != nil {
			log.Fatalf("save state: %v", err)
		}
	}
	slog.Info("announce queued", "message", *message)
}

func runKick(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("kick", flag.ExitOnError)
	playerID := fs.String("player", "", "player id")
	statePath := fs.String("state", "", "snapshot path (optional)")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	_ = fs.Parse(args)
	if *playerID == "" {
		log.Fatal("kick requires -player")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/kick", map[string]interface{}{
			"playerId": *playerID,
		}, nil); err != nil {
			log.Fatalf("kick failed: %v", err)
		}
		slog.Info("kick sent", "player_id", *playerID)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "KICK",
		PlayerID: *playerID,
		Message:  "player kicked",
	})
	if *statePath != "" || *outPath != "" {
		savePath := pickOutPath(*statePath, *outPath)
		if err := saveState(savePath, gameState); err != nil {
			log.Fatalf("save state: %v", err)
		}
	}
	slog.Info("kick logged", "player_id", *playerID)
}

func runBan(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("ban", flag.ExitOnError)
	playerID := fs.String("player", "", "player id")
	reason := fs.String("reason", "", "ban reason")
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	_ = fs.Parse(args)
	if *playerID == "" {
		log.Fatal("ban requires -player")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/ban", map[string]interface{}{
			"playerId": *playerID,
			"reason":   *reason,
		}, nil); err != nil {
			log.Fatalf("ban failed: %v", err)
		}
		slog.Info("ban applied", "player_id", *playerID, "reason", *reason)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	if gameState.Admin == nil {
		gameState.Admin = state.NewAdminState()
	}
	gameState.Admin.Banned[*playerID] = *reason
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "BAN",
		PlayerID: *playerID,
		Message:  "player banned",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("ban applied", "player_id", *playerID, "reason", *reason)
}

func runUnban(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("unban", flag.ExitOnError)
	playerID := fs.String("player", "", "player id")
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	_ = fs.Parse(args)
	if *playerID == "" {
		log.Fatal("unban requires -player")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/unban", map[string]interface{}{
			"playerId": *playerID,
		}, nil); err != nil {
			log.Fatalf("unban failed: %v", err)
		}
		slog.Info("unban applied", "player_id", *playerID)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	if gameState.Admin != nil {
		delete(gameState.Admin.Banned, *playerID)
	}
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "UNBAN",
		PlayerID: *playerID,
		Message:  "player unbanned",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("unban applied", "player_id", *playerID)
}

func runPlayer(args []string, cfg *config.Config, opts cliOptions) {
	if len(args) == 0 {
		log.Fatal("player command required: set-faction|set-level|grant-item")
	}
	switch args[0] {
	case "set-faction":
		playerSetFaction(args[1:], cfg, opts)
	case "set-level":
		playerSetLevel(args[1:], cfg, opts)
	case "grant-item":
		playerGrantItem(args[1:], cfg, opts)
	default:
		log.Fatal("unknown player command")
	}
}

func playerSetFaction(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("player set-faction", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	playerID := fs.String("player", "", "player id")
	faction := fs.String("faction", "", "faction")
	_ = fs.Parse(args)
	if *playerID == "" || *faction == "" {
		log.Fatal("set-faction requires -player -faction")
	}
	if adminClientFrom(cfg, opts) == nil && *statePath == "" {
		log.Fatal("set-faction requires -state in offline mode")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/players/"+*playerID+"/faction", map[string]interface{}{
			"faction": *faction,
		}, nil); err != nil {
			log.Fatalf("set-faction failed: %v", err)
		}
		slog.Info("faction updated", "player_id", *playerID, "faction", strings.ToUpper(*faction))
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	player, ok := gameState.Players.Get(*playerID)
	if !ok {
		log.Fatalf("player not found: %s", *playerID)
	}
	player.Faction = strings.ToUpper(*faction)
	player.UpdatedAt = time.Now()
	gameState.Players.Upsert(player)
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PLAYER",
		PlayerID: player.ID,
		Message:  "faction updated",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("faction updated", "player_id", *playerID, "faction", player.Faction)
}

func playerSetLevel(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("player set-level", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	playerID := fs.String("player", "", "player id")
	level := fs.Int("level", 0, "level")
	_ = fs.Parse(args)
	if *playerID == "" || *level <= 0 {
		log.Fatal("set-level requires -player -level")
	}
	if adminClientFrom(cfg, opts) == nil && *statePath == "" {
		log.Fatal("set-level requires -state in offline mode")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/players/"+*playerID+"/level", map[string]interface{}{
			"level": *level,
		}, nil); err != nil {
			log.Fatalf("set-level failed: %v", err)
		}
		slog.Info("level updated", "player_id", *playerID, "level", *level)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	player, ok := gameState.Players.Get(*playerID)
	if !ok {
		log.Fatalf("player not found: %s", *playerID)
	}
	player.Level = *level
	player.MaxXM = maxXMForLevel(*level)
	if player.XM > player.MaxXM {
		player.XM = player.MaxXM
	}
	player.UpdatedAt = time.Now()
	gameState.Players.Upsert(player)
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PLAYER",
		PlayerID: player.ID,
		Message:  "level updated",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("level updated", "player_id", *playerID, "level", player.Level)
}

func playerGrantItem(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("player grant-item", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	playerID := fs.String("player", "", "player id")
	itemID := fs.String("item", "", "item id")
	amount := fs.Int("amount", 1, "amount")
	_ = fs.Parse(args)
	if *playerID == "" || *itemID == "" || *amount <= 0 {
		log.Fatal("grant-item requires -player -item -amount")
	}
	if adminClientFrom(cfg, opts) == nil && *statePath == "" {
		log.Fatal("grant-item requires -state in offline mode")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/players/"+*playerID+"/grant-item", map[string]interface{}{
			"itemId": *itemID,
			"amount": *amount,
		}, nil); err != nil {
			log.Fatalf("grant-item failed: %v", err)
		}
		slog.Info("item granted", "player_id", *playerID, "item", *itemID, "amount", *amount)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	player, ok := gameState.Players.Get(*playerID)
	if !ok {
		log.Fatalf("player not found: %s", *playerID)
	}
	item := gameplay.NormalizeCubeID(*itemID)
	if _, ok := gameplay.IsKey(item); ok {
		if !gameplay.CanAddKey(player.Inventory, strings.TrimPrefix(item, gameplay.ItemPrefixKey), *amount, player.Level) {
			log.Fatal("key capacity exceeded")
		}
	} else if !gameplay.CanAddItem(player.Inventory, item, *amount, player.Level) {
		log.Fatal("inventory capacity exceeded")
	}
	if player.Inventory == nil {
		player.Inventory = make(map[string]int)
	}
	player.Inventory[item] += *amount
	player.UpdatedAt = time.Now()
	gameState.Players.Upsert(player)
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PLAYER",
		PlayerID: player.ID,
		Message:  "item granted",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("item granted", "player_id", *playerID, "item", item, "amount", *amount)
}

func runPortal(args []string, cfg *config.Config, opts cliOptions) {
	if len(args) == 0 {
		log.Fatal("portal command required: add|update|remove")
	}
	switch args[0] {
	case "add":
		portalAdd(args[1:], cfg, opts)
	case "update":
		portalUpdate(args[1:], cfg, opts)
	case "remove":
		portalRemove(args[1:], cfg, opts)
	default:
		log.Fatal("unknown portal command")
	}
}

func portalAdd(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("portal add", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	id := fs.String("id", "", "portal id")
	title := fs.String("title", "", "portal title")
	coverURL := fs.String("cover-url", "", "portal cover url")
	lat := fs.Float64("lat", math.NaN(), "latitude")
	lon := fs.Float64("lon", math.NaN(), "longitude")
	faction := fs.String("faction", "NEUTRAL", "faction")
	_ = fs.Parse(args)
	if *id == "" || math.IsNaN(*lat) || math.IsNaN(*lon) {
		log.Fatal("portal add requires -id -lat -lon")
	}
	if adminClientFrom(cfg, opts) == nil && *statePath == "" {
		log.Fatal("portal add requires -state in offline mode")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/portals", map[string]interface{}{
			"id":        *id,
			"title":     *title,
			"cover_url": *coverURL,
			"lat":       *lat,
			"lon":       *lon,
			"faction":   *faction,
		}, nil); err != nil {
			log.Fatalf("portal add failed: %v", err)
		}
		slog.Info("portal added", "portal_id", *id)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	portal := state.Portal{
		ID:        *id,
		Title:     strings.TrimSpace(*title),
		CoverURL:  strings.TrimSpace(*coverURL),
		Position:  state.Position{Latitude: *lat, Longitude: *lon},
		Faction:   strings.ToUpper(*faction),
		UpdatedAt: time.Now(),
	}
	if portal.Title == "" {
		portal.Title = portal.ID
	}
	gameState.Portals.Upsert(portal)
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PORTAL",
		PortalID: portal.ID,
		Message:  "portal added",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("portal added", "portal_id", *id)
}

func portalUpdate(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("portal update", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	id := fs.String("id", "", "portal id")
	lat := fs.String("lat", "", "latitude")
	lon := fs.String("lon", "", "longitude")
	title := fs.String("title", "", "portal title")
	coverURL := fs.String("cover-url", "", "portal cover url")
	faction := fs.String("faction", "", "faction")
	level := fs.String("level", "", "level")
	energy := fs.String("energy", "", "energy")
	_ = fs.Parse(args)
	if *id == "" {
		log.Fatal("portal update requires -id")
	}
	if *lat == "" && *lon == "" && *title == "" && *coverURL == "" && *faction == "" && *level == "" && *energy == "" {
		log.Fatal("portal update requires at least one field")
	}
	if adminClientFrom(cfg, opts) == nil && *statePath == "" {
		log.Fatal("portal update requires -state in offline mode")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		payload := map[string]interface{}{}
		if *lat != "" {
			latVal, err := parseFloat(*lat)
			if err != nil {
				log.Fatalf("invalid lat: %v", err)
			}
			payload["lat"] = latVal
		}
		if *lon != "" {
			lonVal, err := parseFloat(*lon)
			if err != nil {
				log.Fatalf("invalid lon: %v", err)
			}
			payload["lon"] = lonVal
		}
		if *faction != "" {
			payload["faction"] = *faction
		}
		if *title != "" {
			payload["title"] = *title
		}
		if *coverURL != "" {
			payload["cover_url"] = *coverURL
		}
		if *level != "" {
			val, err := parseInt(*level)
			if err != nil {
				log.Fatalf("invalid level: %v", err)
			}
			payload["level"] = val
		}
		if *energy != "" {
			val, err := parseInt(*energy)
			if err != nil {
				log.Fatalf("invalid energy: %v", err)
			}
			payload["energy"] = val
		}
		if err := client.doJSON(http.MethodPatch, "/api/v1/admin/portals/"+*id, payload, nil); err != nil {
			log.Fatalf("portal update failed: %v", err)
		}
		slog.Info("portal updated", "portal_id", *id)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	portal, ok := gameState.Portals.Get(*id)
	if !ok {
		log.Fatalf("portal not found: %s", *id)
	}
	if *lat != "" && *lon != "" {
		latVal, err := parseFloat(*lat)
		if err != nil {
			log.Fatalf("invalid lat: %v", err)
		}
		lonVal, err := parseFloat(*lon)
		if err != nil {
			log.Fatalf("invalid lon: %v", err)
		}
		portal.Position = state.Position{Latitude: latVal, Longitude: lonVal}
	}
	if *faction != "" {
		portal.Faction = strings.ToUpper(*faction)
	}
	if *title != "" {
		portal.Title = strings.TrimSpace(*title)
		if portal.Title == "" {
			portal.Title = portal.ID
		}
	}
	if *coverURL != "" {
		portal.CoverURL = strings.TrimSpace(*coverURL)
	}
	if *level != "" {
		val, err := parseInt(*level)
		if err != nil {
			log.Fatalf("invalid level: %v", err)
		}
		portal.Level = val
	}
	if *energy != "" {
		val, err := parseInt(*energy)
		if err != nil {
			log.Fatalf("invalid energy: %v", err)
		}
		portal.Energy = val
	}
	portal.UpdatedAt = time.Now()
	gameState.Portals.Upsert(portal)
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PORTAL",
		PortalID: portal.ID,
		Message:  "portal updated",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("portal updated", "portal_id", *id)
}

func portalRemove(args []string, cfg *config.Config, opts cliOptions) {
	fs := flag.NewFlagSet("portal remove", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	id := fs.String("id", "", "portal id")
	_ = fs.Parse(args)
	if *id == "" {
		log.Fatal("portal remove requires -id")
	}
	if adminClientFrom(cfg, opts) == nil && *statePath == "" {
		log.Fatal("portal remove requires -state in offline mode")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodDelete, "/api/v1/admin/portals/"+*id, nil, nil); err != nil {
			log.Fatalf("portal remove failed: %v", err)
		}
		slog.Info("portal removed", "portal_id", *id)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	gameState.Portals.Remove(*id)
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "PORTAL",
		PortalID: *id,
		Message:  "portal removed",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("portal removed", "portal_id", *id)
}

func runLink(args []string, cfg *config.Config, opts cliOptions) {
	if len(args) == 0 {
		log.Fatal("link command required: remove")
	}
	if args[0] != "remove" {
		log.Fatal("unknown link command")
	}
	fs := flag.NewFlagSet("link remove", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	id := fs.String("id", "", "link id")
	_ = fs.Parse(args[1:])
	if *id == "" {
		log.Fatal("link remove requires -id")
	}
	if adminClientFrom(cfg, opts) == nil && *statePath == "" {
		log.Fatal("link remove requires -state in offline mode")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodDelete, "/api/v1/admin/links/"+*id, nil, nil); err != nil {
			log.Fatalf("link remove failed: %v", err)
		}
		slog.Info("link removed", "link_id", *id)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	gameState.Links.Remove(*id)
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "LINK",
		PortalID: *id,
		Message:  "link removed",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("link removed", "link_id", *id)
}

func runField(args []string, cfg *config.Config, opts cliOptions) {
	if len(args) == 0 {
		log.Fatal("field command required: remove")
	}
	if args[0] != "remove" {
		log.Fatal("unknown field command")
	}
	fs := flag.NewFlagSet("field remove", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	id := fs.String("id", "", "field id")
	_ = fs.Parse(args[1:])
	if *id == "" {
		log.Fatal("field remove requires -id")
	}
	if adminClientFrom(cfg, opts) == nil && *statePath == "" {
		log.Fatal("field remove requires -state in offline mode")
	}
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodDelete, "/api/v1/admin/fields/"+*id, nil, nil); err != nil {
			log.Fatalf("field remove failed: %v", err)
		}
		slog.Info("field removed", "field_id", *id)
		return
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	gameState.Fields.Remove(*id)
	appendLog(gameState, state.LogEntry{
		ID:       uuid.New().String(),
		Type:     "FIELD",
		PortalID: *id,
		Message:  "field removed",
	})
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("field removed", "field_id", *id)
}

func runLogs(args []string, cfg *config.Config, opts cliOptions) {
	if client := adminClientFrom(cfg, opts); client != nil {
		fs := flag.NewFlagSet("logs", flag.ExitOnError)
		limit := fs.Int("limit", 100, "limit")
		_ = fs.Parse(args)
		path := "/api/v1/admin/logs"
		if *limit > 0 {
			path = fmt.Sprintf("%s?limit=%d", path, *limit)
		}
		var data map[string]interface{}
		if err := client.doJSON(http.MethodGet, path, nil, &data); err != nil {
			log.Fatalf("logs failed: %v", err)
		}
		slog.Info("logs", "data", data)
		return
	}
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	limit := fs.Int("limit", 100, "limit")
	_ = fs.Parse(args)
	if *statePath == "" {
		log.Fatal("logs requires -state")
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	logs := []state.LogEntry{}
	if gameState.Logs != nil {
		logs = gameState.Logs.List()
	}
	if *limit > 0 && len(logs) > *limit {
		logs = logs[len(logs)-*limit:]
	}
	for _, entry := range logs {
		slog.Info("log",
			"id", entry.ID,
			"type", entry.Type,
			"player", entry.PlayerID,
			"portal", entry.PortalID,
			"mu", entry.MU,
			"message", entry.Message,
			"ts", entry.Timestamp,
		)
	}
}

func runGCLogs(args []string, cfg *config.Config, opts cliOptions) {
	if client := adminClientFrom(cfg, opts); client != nil {
		fs := flag.NewFlagSet("gc-logs", flag.ExitOnError)
		keep := fs.Int("keep", 100, "keep last N logs")
		_ = fs.Parse(args)
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/gc-logs", map[string]interface{}{
			"keep": *keep,
		}, nil); err != nil {
			log.Fatalf("gc-logs failed: %v", err)
		}
		slog.Info("gc-logs completed", "kept", *keep)
		return
	}
	fs := flag.NewFlagSet("gc-logs", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	keep := fs.Int("keep", 100, "keep last N logs")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	_ = fs.Parse(args)
	if *statePath == "" {
		log.Fatal("gc-logs requires -state")
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	if gameState.Logs == nil {
		return
	}
	logs := gameState.Logs.List()
	if *keep > 0 && len(logs) > *keep {
		logs = logs[len(logs)-*keep:]
	}
	gameState.Logs = state.NewLogStore(len(logs))
	for _, entry := range logs {
		gameState.Logs.Add(entry)
	}
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("gc-logs completed", "kept", len(logs))
}

func runRecalcStats(args []string, cfg *config.Config, opts cliOptions) {
	if client := adminClientFrom(cfg, opts); client != nil {
		if err := client.doJSON(http.MethodPost, "/api/v1/admin/recalc-stats", nil, nil); err != nil {
			log.Fatalf("recalc-stats failed: %v", err)
		}
		slog.Info("recalc-stats completed")
		return
	}
	fs := flag.NewFlagSet("recalc-stats", flag.ExitOnError)
	statePath := fs.String("state", "", "snapshot path")
	outPath := fs.String("out", "", "output snapshot path (optional)")
	_ = fs.Parse(args)
	if *statePath == "" {
		log.Fatal("recalc-stats requires -state")
	}
	gameState, err := loadState(*statePath, cfg)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	players := gameState.Players.List()
	for _, player := range players {
		player.PortalCaptures = 0
		player.LinksCreated = 0
		player.FieldsCreated = 0
		player.MUTotal = 0
		player.AP = 0
		gameState.Players.Upsert(player)
	}
	logs := []state.LogEntry{}
	if gameState.Logs != nil {
		logs = gameState.Logs.List()
	}
	for _, entry := range logs {
		player, ok := gameState.Players.Get(entry.PlayerID)
		if !ok {
			continue
		}
		switch entry.Type {
		case "CAPTURE":
			player.PortalCaptures++
			player.AP += 500
		case "LINK":
			player.LinksCreated++
			player.AP += 313
		case "FIELD":
			player.FieldsCreated++
			player.MUTotal += entry.MU
			player.AP += 1250
		}
		gameState.Players.Upsert(player)
	}
	savePath := pickOutPath(*statePath, *outPath)
	if err := saveState(savePath, gameState); err != nil {
		log.Fatalf("save state: %v", err)
	}
	slog.Info("recalc-stats completed")
}

func runReloadConfig(args []string, cfg *config.Config, opts cliOptions) {
	client := adminClientFrom(cfg, opts)
	if client == nil {
		slog.Info("reload-config noop in offline CLI")
		return
	}
	if err := client.doJSON(http.MethodPost, "/api/v1/admin/reload-config", nil, nil); err != nil {
		log.Fatalf("reload-config failed: %v", err)
	}
	slog.Info("reload-config completed")
}

func runRotateLogs(args []string, cfg *config.Config, opts cliOptions) {
	client := adminClientFrom(cfg, opts)
	if client == nil {
		slog.Info("rotate-logs noop in offline CLI")
		return
	}
	if err := client.doJSON(http.MethodPost, "/api/v1/admin/rotate-logs", nil, nil); err != nil {
		log.Fatalf("rotate-logs failed: %v", err)
	}
	slog.Info("rotate-logs completed")
}

func loadState(path string, cfg *config.Config) (*state.GameState, error) {
	if path == "" {
		gs := state.NewGameState()
		if cfg.Gameplay.LogCapacity > 0 {
			gs.Logs = state.NewLogStore(cfg.Gameplay.LogCapacity)
		}
		return gs, nil
	}
	snapshot, err := state.LoadFromFile(path)
	if err != nil {
		return nil, err
	}
	gs := state.NewGameStateFromSnapshot(snapshot)
	if cfg.Gameplay.LogCapacity > 0 && gs.Logs != nil {
		gs.Logs = state.NewLogStore(cfg.Gameplay.LogCapacity)
		for _, entry := range snapshot.Logs {
			gs.Logs.Add(entry)
		}
	}
	return gs, nil
}

func saveState(path string, gameState *state.GameState) error {
	if path == "" {
		return nil
	}
	return state.SaveToFile(path, gameState.Snapshot())
}

type adminClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func adminClientFrom(cfg *config.Config, opts cliOptions) *adminClient {
	if opts.apiBase == "" {
		return nil
	}
	token := opts.token
	if token == "" {
		token = cfg.Admin.Token
	}
	return &adminClient{
		baseURL: opts.apiBase,
		token:   token,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *adminClient) doJSON(method, path string, body interface{}, out interface{}) error {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequest(method, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Admin-Token", c.token)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var apiResp api.Response
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}
	if apiResp.Errno != api.ErrCodeSuccess {
		return fmt.Errorf("admin api error: %s", apiResp.Errmsg)
	}
	if out != nil {
		raw, err := json.Marshal(apiResp.Data)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, out); err != nil {
			return err
		}
	}
	return nil
}

func pickOutPath(statePath, outPath string) string {
	if outPath != "" {
		return outPath
	}
	return statePath
}

func appendLog(gameState *state.GameState, entry state.LogEntry) {
	if gameState == nil || gameState.Logs == nil {
		return
	}
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	gameState.Logs.Add(entry)
}

func parseFloat(raw string) (float64, error) {
	return strconvParseFloat(strings.TrimSpace(raw))
}

func parseInt(raw string) (int, error) {
	return strconvParseInt(strings.TrimSpace(raw))
}

func strconvParseFloat(raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func strconvParseInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func maxXMForLevel(level int) int {
	switch {
	case level <= 1:
		return 3000
	case level == 2:
		return 3500
	case level == 3:
		return 4000
	case level == 4:
		return 4500
	case level == 5:
		return 5000
	case level == 6:
		return 5500
	case level == 7:
		return 6000
	case level == 8:
		return 6500
	case level == 9:
		return 7000
	case level == 10:
		return 7500
	case level == 11:
		return 8000
	case level == 12:
		return 8500
	case level == 13:
		return 9000
	case level == 14:
		return 9500
	default:
		return 10000
	}
}
