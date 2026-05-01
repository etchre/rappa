package com.rappa.premium;

import com.sedmelluq.discord.lavaplayer.tools.FriendlyException;
import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

@Component
public class YtDlpResolver {
    private static final Logger log = LoggerFactory.getLogger(YtDlpResolver.class);

    private final RappaPremiumConfig config;

    public YtDlpResolver(RappaPremiumConfig config) {
        this.config = config;
    }

    public String resolveStreamUrl(String identifier) {
        List<String> command = new ArrayList<>();
        command.add("yt-dlp");
        command.add("--no-progress");
        command.add("--format");
        command.add(config.format());
        command.add("--get-url");

        if (!config.cookiesFile().isBlank()) {
            command.add("--cookies");
            command.add(config.cookiesFile());
        }

        if (!config.extractorArgs().isBlank()) {
            command.add("--extractor-args");
            command.add(config.extractorArgs());
        }

        command.add(identifier);

        ProcessBuilder processBuilder = new ProcessBuilder(command);
        processBuilder.redirectErrorStream(true);

        try {
            log.info("Resolving premium stream with yt-dlp identifier={}", identifier);
            long startTime = System.nanoTime();
            Process process = processBuilder.start();
            CompletableFuture<String> outputFuture = CompletableFuture.supplyAsync(() -> readOutput(process));

            boolean exited = process.waitFor(config.timeoutMillis(), TimeUnit.MILLISECONDS);
            if (!exited) {
                process.destroyForcibly();
                throw new FriendlyException(
                    "yt-dlp timed out while resolving the premium stream.",
                    FriendlyException.Severity.SUSPICIOUS,
                    null
                );
            }

            String output = outputFuture.join();
            int exitCode = process.exitValue();
            if (exitCode != 0) {
                throw new FriendlyException(
                    "yt-dlp failed while resolving the premium stream.",
                    FriendlyException.Severity.SUSPICIOUS,
                    new RuntimeException(output)
                );
            }

            String streamUrl = firstNonEmptyLine(output);
            if (streamUrl == null) {
                throw new FriendlyException(
                    "yt-dlp returned an empty premium stream URL.",
                    FriendlyException.Severity.SUSPICIOUS,
                    null
                );
            }

            long elapsedMillis = Duration.ofNanos(System.nanoTime() - startTime).toMillis();
            log.info("Resolved premium stream with yt-dlp in {}ms", elapsedMillis);
            return streamUrl;
        } catch (IOException e) {
            throw new FriendlyException(
                "Failed to start yt-dlp. Confirm yt-dlp is installed in the Lavalink container.",
                FriendlyException.Severity.FAULT,
                e
            );
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new FriendlyException(
                "Interrupted while resolving the premium stream.",
                FriendlyException.Severity.SUSPICIOUS,
                e
            );
        }
    }

    private static String readOutput(Process process) {
        StringBuilder output = new StringBuilder();
        try (BufferedReader reader = new BufferedReader(new InputStreamReader(process.getInputStream(), StandardCharsets.UTF_8))) {
            String line;
            while ((line = reader.readLine()) != null) {
                output.append(line).append('\n');
            }
        } catch (IOException e) {
            output.append("failed to read yt-dlp output: ").append(e.getMessage()).append('\n');
        }
        return output.toString();
    }

    private static String firstNonEmptyLine(String output) {
        for (String line : output.split("\\R")) {
            String trimmed = line.trim();
            if (!trimmed.isEmpty() && trimmed.startsWith("http")) {
                return trimmed;
            }
        }
        return null;
    }
}
