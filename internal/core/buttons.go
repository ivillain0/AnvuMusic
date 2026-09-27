/*
 * ● AnvuMusic
 * ○ A high-performance engine for streaming music in Telegram voicechats.
 *
 * Copyright (C) 2026 Team Echo
 */

package core

import (
	"fmt"

	tg "github.com/amarnathcjd/gogram/telegram"

	"main/internal/config"
	"main/internal/database"
	"main/internal/locales"
	"main/internal/utils"
)

var F func(chatID int64, key string, values ...locales.Arg) string // overwritten from main.go

func AddMeMarkup(chatID int64) tg.ReplyMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.URL(
				F(chatID, "ADD_ME_BTN"),
				"https://t.me/"+Bot.Me().Username+"?startgroup&admin=invite_users",
			).Success(),
		).
		Build()
}

func GetCancelKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.Data(F(chatID, "DOWNLOAD_CANCEL_BTN"), "cancel").Danger(),
		).
		Build()
}

func GetBroadcastCancelKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.Data(F(chatID, "BROADCAST_CANCEL_BTN"), "bcast_cancel").Danger(),
		).
		Build()
}

func SuppMarkup(chatID int64) tg.ReplyMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.URL(F(chatID, "SUPPORT_BTN"), config.SupportChat).Primary(),
		).
		Build()
}

func GetStopConfirmMarkup(
	chatID int64,
	r *RoomState,
	isPaused bool,
) tg.ReplyMarkup {
	btn := tg.NewKeyboard()
	prefix := "room:"
	if r.ChannelPlayID() != 0 {
		prefix = "croom:"
	}

	if isPaused {
		btn.AddRow(
			tg.Button.Data(F(chatID, "CONFIRM_RESUME_BTN"), prefix+"resume").Success(),
		)
	} else {
		btn.AddRow(
			tg.Button.Data(F(chatID, "CONFIRM_UNMUTE_BTN"), prefix+"unmute").Success(),
		)
	}

	btn.AddRow(
		tg.Button.Data(F(chatID, "CONFIRM_STOP_BTN"), prefix+"stop").Danger(),
	)

	return btn.Build()
}

func GetPlayMarkup(chatID int64, r *RoomState, queued bool) tg.ReplyMarkup {
	btn := tg.NewKeyboard()
	prefix := "room:"
	if r.ChannelPlayID() != 0 {
		prefix = "croom:"
	}
	track := r.Track()
	duration := 0
	if track != nil {
		duration = track.Duration
	}

	// Progress bar row (only when not queued)
	if !queued {
		progress := utils.GetProgressBar(r.Position(), duration)
		progress = formatDuration(
			r.Position(),
		) + " " + progress + " " + formatDuration(
			duration,
		)
		btn.AddRow(
			tg.Button.Data(progress, "progress"),
		)
	}

	// Row 1: Playback controls — Resume, Pause, Replay, Skip, Stop
	btn.AddRow(
		tg.Button.Data("▷", prefix+"resume").Success(),
		tg.Button.Data("II", prefix+"pause").Danger(),
		tg.Button.Data("⟳", prefix+"replay").Primary(),
		tg.Button.Data("‣‣I", prefix+"skip").Primary(),
		tg.Button.Data("▢", prefix+"stop").Danger(),
	)

	// Row 2: Seek back, Settings (⚙️), Seek forward
	btn.AddRow(
		tg.Button.Data("↩ 15s", prefix+"seekback_15").Primary(),
		tg.Button.Data("Pl+", prefix+"playlist").Primary(),
		tg.Button.Data("15s ↪", prefix+"seek_15").Primary(),
	)

	// Row 3: Autoplay toggle
	autoplayBtn := F(chatID, "AUTOPLAY_BTN_OFF")
	if r.Autoplay() {
		autoplayBtn = F(chatID, "AUTOPLAY_BTN_ON")
	}
	btn.AddRow(
		tg.Button.Data(autoplayBtn, prefix+"autoplay").Primary(),
	)

	// Row 4: Close only
	btn.AddRow(
		tg.Button.Data(F(chatID, "CLOSE_BTN"), "close").Danger(),
	)

	return btn.Build()
}

// GetPlaybackSettingsMarkup builds the ⚙️ settings view shown when the settings
// button is tapped on the now-playing panel: the autoplay + playlist actions
// and a Back button that returns to the normal control panel. The view
// auto-closes (single edit) after SettingsViewWindow seconds.
func GetPlaybackSettingsMarkup(chatID int64, r *RoomState) tg.ReplyMarkup {
	btn := tg.NewKeyboard()
	prefix := "room:"
	if r.ChannelPlayID() != 0 {
		prefix = "croom:"
	}

	autoplayBtn := F(chatID, "AUTOPLAY_BTN_OFF")
	if r.Autoplay() {
		autoplayBtn = F(chatID, "AUTOPLAY_BTN_ON")
	}
	btn.AddRow(
		tg.Button.Data(autoplayBtn, prefix+"autoplay").Primary(),
		tg.Button.Data(F(chatID, "ADD_TO_PLAYLIST_BTN"), prefix+"playlist").Primary(),
	)

	btn.AddRow(
		tg.Button.Data(F(chatID, "PLAYBACK_BACK_BTN"), prefix+"back").Primary(),
	)

	return btn.Build()
}

// GetPlaylistPickerMarkup builds the playlist chooser keyboard shown after
// tapping the ➕ Playlist button on the now-playing panel.
func GetPlaylistPickerMarkup(
	chatID int64,
	playlists []database.Playlist,
) tg.ReplyMarkup {
	kb := tg.NewKeyboard()

	if len(playlists) == 0 {
		kb.AddRow(tg.Button.Data(F(chatID, "PLAYLIST_CREATE_SAVE_BTN"), "plist:create").Success())
	} else {
		for _, pl := range playlists {
			kb.AddRow(tg.Button.Data(pl.Name, "plist:"+pl.ID).Primary())
		}
	}

	kb.AddRow(tg.Button.Data(F(chatID, "CLOSE_BTN"), "close").Danger())

	return kb.Build()
}

// GetSettingsMarkup builds the per-chat settings panel keyboard.
func GetSettingsMarkup(
	chatID int64,
	playMode bool,
	adminMode string,
	cmdDelete bool,
) tg.ReplyMarkup {
	kb := tg.NewKeyboard()

	playBtn := F(chatID, "SETTINGS_PLAY_EVERYONE_BTN")
	if playMode {
		playBtn = F(chatID, "SETTINGS_PLAY_ADMINS_BTN")
	}
	kb.AddRow(tg.Button.Data(playBtn, "settings:play").Primary())

	adminBtn := F(chatID, "SETTINGS_ADMIN_ADMINS_BTN")
	if adminMode == "everyone" {
		adminBtn = F(chatID, "SETTINGS_ADMIN_EVERYONE_BTN")
	}
	kb.AddRow(tg.Button.Data(adminBtn, "settings:admin").Primary())

	deleteBtn := F(chatID, "SETTINGS_DELETE_OFF_BTN")
	if cmdDelete {
		deleteBtn = F(chatID, "SETTINGS_DELETE_ON_BTN")
	}
	kb.AddRow(tg.Button.Data(deleteBtn, "settings:delete").Primary())

	kb.AddRow(tg.Button.Data(F(chatID, "CLOSE_BTN"), "close").Danger())

	return kb.Build()
}

func GetGroupHelpKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	bot := "https://t.me/" + Bot.Me().Username
	return tg.NewKeyboard().
		AddRow(
			tg.Button.URL(F(chatID, "GC_HELP_BTN"), bot+"?start=pm_help").Primary(),
			tg.Button.URL(F(chatID, "GC_UPDATES_BTN"), config.SupportChannel).Primary(),
		).
		Build()
}

func GetStartMarkup(chatID int64) tg.ReplyMarkup {
	bot := "https://t.me/" + Bot.Me().Username
	kb := tg.NewKeyboard()

	// Row 1: Add to group
	kb.AddRow(
		tg.Button.URL(F(chatID, "ADD_ME_BTN"), bot+"?startgroup&admin=invite_users").Success(),
	)

	// Row 2: Support (opens support panel), Language
	kb.AddRow(
		tg.Button.Data(F(chatID, "SUPPORT_BTN"), "support_panel").Primary(),
		tg.Button.Data(F(chatID, "LANGUAGE_BTN"), "lang").Primary(),
	)

	// Row 3: Help
	kb.AddRow(
		tg.Button.Data(F(chatID, "HELP_BTN"), "help_cb").Primary(),
	)

	return kb.Build()
}

func GetSupportMarkup(chatID int64) tg.ReplyMarkup {
	kb := tg.NewKeyboard()

	// Row 1 (2×2 grid row 1): Support Group, Updates Channel
	kb.AddRow(
		tg.Button.URL(F(chatID, "SUPPORT_BTN"), config.SupportChat).Primary(),
		tg.Button.URL(F(chatID, "UPDATES_BTN"), config.SupportChannel).Primary(),
	)

	// Row 2 (2×2 grid row 2): Owner, Source
	if config.OwnerID != 0 {
		kb.AddRow(
			tg.Button.URL(F(chatID, "OWNER_BTN"), "tg://user?id="+utils.IntToStr(config.OwnerID)).Primary(),
			tg.Button.URL(F(chatID, "SOURCE_BTN"), "https://t.me/lelobhaisource").Primary(),
		)
	} else {
		kb.AddRow(
			tg.Button.URL(F(chatID, "SOURCE_BTN"), "https://t.me/lelobhaisource").Primary(),
		)
	}

	// Row 3: Back to home & Close
	kb.AddRow(
		tg.Button.Data(F(chatID, "HELP_HOME_PANEL_BTN"), "start").Primary(),
		tg.Button.Data(F(chatID, "CLOSE_BTN"), "close").Danger(),
	)

	return kb.Build()
}

func GetHelpKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.Data(
				F(chatID, "HELP_PUBLIC_BTN"),
				"help:public",
			).Primary(),
			tg.Button.Data(
				F(chatID, "HELP_ADMINS_BTN"),
				"help:admins",
			).Primary(),
		).
		AddRow(
			tg.Button.Data(
				F(chatID, "HELP_OWNER_BTN"),
				"help:owner",
			).Primary(),
			tg.Button.Data(
				F(chatID, "HELP_SUDOERS_BTN"),
				"help:sudoers",
			).Primary(),
		).
		AddRow(
			tg.Button.Data(
				F(chatID, "HELP_HOME_PANEL_BTN"),
				"start",
			).Primary(),
		).
		Build()
}

func GetBackKeyboard(chatID int64) *tg.ReplyInlineMarkup {
	return tg.NewKeyboard().
		AddRow(
			tg.Button.Data(
				F(chatID, "HELP_BACK_CATEGORIES_BTN"),
				"help:main",
			).Primary(),
			tg.Button.Data(
				F(chatID, "HELP_HOME_PANEL_BTN"),
				"start",
			).Primary(),
		).
		Build()
}

func formatDuration(sec int) string {
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s) // HH:MM:SS
	}
	return fmt.Sprintf("%02d:%02d", m, s) // MM:SS
}
