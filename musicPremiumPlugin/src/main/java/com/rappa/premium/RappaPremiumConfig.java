package com.rappa.premium;

import org.springframework.stereotype.Component;

@Component
public class RappaPremiumConfig {
    public static final String SOURCE_NAME = "rappa-premium";
    public static final String IDENTIFIER_PREFIX = SOURCE_NAME + ":";

    private final String cookiesFile;
    private final String extractorArgs;
    private final String format;
    private final long timeoutMillis;

    public RappaPremiumConfig() {
        this.cookiesFile = env("YTDLP_COOKIES_FILE", "");
        this.extractorArgs = env("YTDLP_EXTRACTOR_ARGS", "youtube:player_client=web_music");
        this.format = env("YTDLP_FORMAT", "bestaudio/best");
        this.timeoutMillis = parseLong(env("YTDLP_TIMEOUT_MS", "30000"), 30000L);
    }

    public String cookiesFile() {
        return cookiesFile;
    }

    public String extractorArgs() {
        return extractorArgs;
    }

    public String format() {
        return format;
    }

    public long timeoutMillis() {
        return timeoutMillis;
    }

    private static String env(String name, String defaultValue) {
        String value = System.getenv(name);
        if (value == null || value.isBlank()) {
            return defaultValue;
        }
        return value;
    }

    private static long parseLong(String value, long defaultValue) {
        try {
            return Long.parseLong(value);
        } catch (NumberFormatException ignored) {
            return defaultValue;
        }
    }
}
