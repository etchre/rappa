package com.rappa.premium;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.context.annotation.Configuration;

@Configuration
public class RappaPremiumPlugin {
    private static final Logger log = LoggerFactory.getLogger(RappaPremiumPlugin.class);

    public RappaPremiumPlugin() {
        log.info("Rappa premium plugin loaded");
    }
}
