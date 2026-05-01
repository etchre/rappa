package com.rappa.premium;

import com.sedmelluq.discord.lavaplayer.player.AudioPlayerManager;
import com.sedmelluq.discord.lavaplayer.source.AudioSourceManager;
import com.sedmelluq.discord.lavaplayer.tools.FriendlyException;
import com.sedmelluq.discord.lavaplayer.track.AudioItem;
import com.sedmelluq.discord.lavaplayer.track.AudioReference;
import com.sedmelluq.discord.lavaplayer.track.AudioTrack;
import com.sedmelluq.discord.lavaplayer.track.AudioTrackInfo;
import java.io.DataInput;
import java.io.DataOutput;
import java.io.IOException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;

@Service
public class RappaPremiumSourceManager implements AudioSourceManager {
    private static final Logger log = LoggerFactory.getLogger(RappaPremiumSourceManager.class);

    private final YtDlpResolver resolver;

    public RappaPremiumSourceManager(YtDlpResolver resolver) {
        this.resolver = resolver;
    }

    @Override
    public String getSourceName() {
        return RappaPremiumConfig.SOURCE_NAME;
    }

    @Override
    public AudioItem loadItem(AudioPlayerManager manager, AudioReference reference) {
        String identifier = reference.identifier;
        if (identifier == null || !identifier.startsWith(RappaPremiumConfig.IDENTIFIER_PREFIX)) {
            return null;
        }

        String originalIdentifier = identifier.substring(RappaPremiumConfig.IDENTIFIER_PREFIX.length());
        if (originalIdentifier.isBlank()) {
            return null;
        }

        String streamUrl = resolver.resolveStreamUrl(originalIdentifier);
        AudioItem item = manager.loadItemSync(new AudioReference(streamUrl, reference.title));
        if (item == null) {
            throw new FriendlyException(
                "yt-dlp resolved the premium stream, but Lavalink could not load the direct stream URL.",
                FriendlyException.Severity.SUSPICIOUS,
                null
            );
        }

        log.info("Loaded premium identifier via delegated stream URL");
        return item;
    }

    @Override
    public boolean isTrackEncodable(AudioTrack track) {
        return false;
    }

    @Override
    public void encodeTrack(AudioTrack track, DataOutput output) throws IOException {
        // Approach A delegates to Lavalink's existing HTTP source, so this source creates no custom tracks to encode.
    }

    @Override
    public AudioTrack decodeTrack(AudioTrackInfo trackInfo, DataInput input) throws IOException {
        return null;
    }

    @Override
    public void shutdown() {
    }
}
