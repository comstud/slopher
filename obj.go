package slopher

import "encoding/json"
import "time"
import "strconv"

// Custom Time so we can deserialize epoch from int or string
type EpochTime time.Time

func (t *EpochTime) UnmarshalJSON(b []byte) (err error) {
    var sec int64

    if err = json.Unmarshal(b, &sec); err != nil {
        var str string

        if err = json.Unmarshal(b, &str); err != nil {
            return
        }

        sec, err = strconv.ParseInt(str, 10, 64)
        if err != nil {
            return
        }
    }

    *t = EpochTime(time.Unix(sec, 0))

    return
}

type BotIcons struct {
    Image48            string          `json:"image_48"`
}

type Bot struct {
    Deleted            bool            `json:"deleted"`
    Icons             *BotIcons        `json:"icons,omitempty"`
    ID                 string          `json:"id"`
    Name               string          `json:"name"`
}
type ChannelTopic struct {
    Creator            string          `json:"creator"`
    LastSet            EpochTime       `json:"last_set"`
    Value              string          `json:"value"`
}

type ChannelPurpose struct {
    Creator            string          `json:"creator"`
    LastSet            EpochTime       `json:"last_set"`
    Value              string          `json:"value"`
}

// Used for both 'channel' and 'im' json objects.
type Channel struct {
    Created            EpochTime       `json:"created"`
    ID                 string          `json:"id"`
    LastRead           string          `json:"last_read"`
    Latest             *Message        `json:"latest,omitempty"`
    UnreadCount        uint            `json:"unread_count"`
    UnreadCountDisplay uint            `json:"unread_count_display"`

    // Attributes returned in "channel" from rtm.start
    Creator            string          `json:"creator"`
    IsArchived         bool            `json:"is_archived"`
    IsChannel          bool            `json:"is_channel"`
    IsGeneral          bool            `json:"is_general"`
    IsMember           bool            `json:"is_member"`
    Name               string          `json:"name"`
    Members            []string        `json:"members"`
    Purpose            *ChannelPurpose `json:"purpose,omitempty"`
    Topic              *ChannelTopic   `json:"topic,omitempty"`

    // Attributes returned in "im" from rtm.start
    UserID             string          `json:"user"`
    IsIM               bool            `json:"is_im"`
    IsOpen             bool            `json:"is_open"`
}

type Group struct {
}

type ImageAttachment struct {
    Title           string            `json:"title"`
    TitleLink       string            `json:"title_link"`
    ImageURL        string            `json:"image_url"`
    ImageWidth      int               `json:"image_width"`
    ImageHeight     int               `json:"image_height"`
    ImageBytes      int               `json:"image_bytes"`  
}

type TwitterAttachment struct {
    Id              int               `json:"id"`
    AuthorLink      string            `json:"author_link"`
    Pretext         string            `json:"pretext"`
    Fallback        string            `json:"fallback"`
    AuthorName      string            `json:"author_name"`
    AuthorIcon      string            `json:"author_icon"`
    AuthorSubname   string            `json:"author_subname"`
    ServerName      string            `json:"service_name"`
}

type Attachment struct {
    Text            string            `json:"text"`
                    *ImageAttachment
                    *TwitterAttachment
    /* ... */
}

type MessageEditedInfo struct {
    UserID          string            `json:"user"`
    TimeStamp       string            `json:"ts"`
}

type Message struct {
    TimeStamp       string            `json:"ts"`
    UserID          string            `json:"user"`
    ChannelID       string            `json:"channel"`
    Text            string            `json:"text"`
    SubType         string            `json:"subtype"`
    SubMessage     *Message           `json:"message,omitempty"`
    Attachments   []Attachment        `json:"attachments"`
    Edited         *MessageEditedInfo `json:"edited"`
}

type TeamPrefs struct {
    AllowMessageDeletion bool      `json:"allow_message_deletion"`
/*
            "compliance_export_start": 0,
            "default_channels": [
                "<CHANNEL_ID>"
            ],
            "display_real_names": true,
            "dm_retention_duration": 0,
            "dm_retention_type": 0,
            "group_retention_duration": 0,
            "group_retention_type": 0,
            "hide_referers": true,
            "msg_edit_window_mins": -1,
            "require_at_for_mention": 0,
            "retention_duration": 0,
            "retention_type": 0,
            "warn_before_at_channel": "always",
            "who_can_archive_channels": "regular",
            "who_can_at_channel": "ra",
            "who_can_at_everyone": "regular",
            "who_can_create_channels": "regular",
            "who_can_create_groups": "ra",
            "who_can_kick_channels": "admin",
            "who_can_kick_groups": "regular",
            "who_can_post_general": "ra"
*/
}

type TeamIcons struct {
    Image102             string    `json:"image_102"`
    Image132             string    `json:"image_132"`
    Image34              string    `json:"image_34"`
    Image44              string    `json:"image_44"`
    Image68              string    `json:"image_68"`
    Image88              string    `json:"image_88"`
    ImageDefault         bool      `json:"image_default"`
}

type Team struct {
    Domain               string    `json:"domain"`
    EmailDomain          string    `json:"email_domain"`
    Icon                *TeamIcons `json:"icon,omitempty"`
    ID                   string    `json:"id"`
    MsgEditWindowMins    int       `json:"msg_edit_window_mins"`
    Name                 string    `json:"name"`
    OverStorageLimit     bool      `json:"over_storage_limit"`
    Prefs               *TeamPrefs `json:"prefs,omitempty"`
}

type SelfPrefs struct {
/*
            "all_channels_loud": false,
            "arrow_history": false,
            "at_channel_suppressed_channels": "",
            "autoplay_chat_sounds": true,
            "collapsible": false,
            "collapsible_by_click": true,
            "color_names_in_list": true,
            "comma_key_prefs": false,
            "convert_emoticons": true,
            "display_real_names_override": 0,
            "dropbox_enabled": false,
            "email_alerts": "instant",
            "email_alerts_sleep_until": 0,
            "email_compact_header": false,
            "email_misc": true,
            "email_weekly": true,
            "emoji_autocomplete_big": false,
            "emoji_mode": "default",
            "enable_flexpane_rework": false,
            "enter_is_special_in_tbt": false,
            "expand_inline_imgs": true,
            "expand_internal_inline_imgs": true,
            "expand_non_media_attachments": true,
            "expand_snippets": false,
            "f_key_search": false,
            "flex_resize_window": false,
            "full_text_extracts": false,
            "fuller_timestamps": false,
            "fuzzy_matching": false,
            "graphic_emoticons": false,
            "growls_enabled": true,
            "has_created_channel": false,
            "has_invited": false,
            "has_uploaded": false,
            "highlight_words": "",
            "k_key_omnibox": true,
            "last_seen_at_channel_warning": 0,
            "last_snippet_type": "",
            "load_lato_2": false,
            "loud_channels": "",
            "loud_channels_set": "",
            "ls_disabled": false,
            "mac_speak_speed": 250,
            "mac_speak_voice": "com.apple.speech.synthesis.voice.Alex",
            "mac_ssb_bounce": "",
            "mac_ssb_bullet": true,
            "mark_msgs_read_immediately": true,
            "messages_theme": "default",
            "msg_preview": false,
            "msg_preview_displaces": true,
            "msg_preview_persistent": true,
            "mute_sounds": false,
            "muted_channels": "",
            "never_channels": "",
            "new_msg_snd": "knock_brush.mp3",
            "no_created_overlays": false,
            "no_joined_overlays": false,
            "no_macssb1_banner": false,
            "no_text_in_notifications": false,
            "no_winssb1_banner": false,
            "obey_inline_img_limit": true,
            "pagekeys_handled": true,
            "posts_formatting_guide": true,
            "privacy_policy_seen": true,
            "prompted_for_email_disabling": false,
            "push_at_channel_suppressed_channels": "",
            "push_dm_alert": true,
            "push_everything": false,
            "push_idle_wait": 2,
            "push_loud_channels": "",
            "push_loud_channels_set": "",
            "push_mention_alert": true,
            "push_mention_channels": "",
            "push_sound": "b2.mp3",
            "require_at": true,
            "search_exclude_bots": false,
            "search_exclude_channels": "",
            "search_only_my_channels": false,
            "search_sort": "timestamp",
            "seen_channel_menu_tip_card": false,
            "seen_channels_tip_card": false,
            "seen_domain_invite_reminder": false,
            "seen_flexpane_tip_card": false,
            "seen_member_invite_reminder": false,
            "seen_message_input_tip_card": false,
            "seen_search_input_tip_card": false,
            "seen_ssb_prompt": false,
            "seen_team_menu_tip_card": false,
            "seen_user_menu_tip_card": false,
            "seen_welcome_2": false,
            "show_member_presence": true,
            "show_typing": true,
            "sidebar_behavior": "",
            "sidebar_theme": "default",
            "sidebar_theme_custom_values": "",
            "snippet_editor_wrap_long_lines": false,
            "speak_growls": false,
            "ss_emojis": true,
            "start_scroll_at_oldest": true,
            "tab_ui_return_selects": true,
            "time24": false,
            "tz": null,
            "user_colors": "",
            "webapp_spellcheck": true,
            "welcome_message_hidden": false,
            "winssb_run_from_tray": true
*/
}

type Self struct {
    Created            EpochTime   `json:"created"`
    ID                 string      `json:"id"`
    ManualPresence     string      `json:"manual_presence"`
    Name               string      `json:"name"`
    Prefs              *SelfPrefs  `json:"prefs,omitempty"`
}

type UserProfile struct {
    Email              string      `json:"email"`
    FirstName          string      `json:"first_key"`
    Image_192          string      `json:"image_192"`
    Image_24           string      `json:"image_24"`
    Image_32           string      `json:"image_32"`
    Image_48           string      `json:"image_48"`
    Image_72           string      `json:"image_72"`
    ImageOriginal      string      `json:"image_original"`
    LastName           string      `json:"last_key"`
    Phone              string      `json:"phone"`
    RealName           string      `json:"real_name"`
    RealNameNormalized string      `json:"real_name_normalized"`
    Skype              string      `json:"skype"`
    Title              string      `json:"title"`
}

type User struct {
    Color             string       `json:"color"`
    Deleted           bool         `json:"deleted"`
    HasFiles          bool         `json:"has_files"`
    ID                string       `json:"id"`
    IsAdmin           bool         `json:"is_admin"`
    IsBot             bool         `json:"is_bot"`
    IsOwner           bool         `json:"is_owner"`
    IsPrimaryOwner    bool         `json:"is_primary_owner"`
    IsRestricted      bool         `json:"is_restricted"`
    IsUltraRestricted bool         `json:"is_ultra_restricted"`
    Name              string       `json:"name"`
    Presence          string       `json:"presence"`
    Profile           *UserProfile `json:"profile,omitempty"`
    RealName          string       `json:"real_name"`
    Status            string       `json:"status"`
    TimeZone          string       `json:"tz"`
    TimeZoneLabel     string       `json:"tz_label"`
    TimeZoneOffSet    int          `json:"tz_offset"`
}
