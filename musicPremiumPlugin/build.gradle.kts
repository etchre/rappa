plugins {
    java
    id("dev.arbjerg.lavalink.gradle-plugin") version "1.0.15"
}

group = "com.rappa"
version = "0.1.0"

dependencies {
    compileOnly("dev.arbjerg.lavalink:plugin-api:4.2.1")
    compileOnly("dev.arbjerg:lavaplayer:2.2.6")
    compileOnly("org.springframework:spring-context:6.1.15")
    compileOnly("org.slf4j:slf4j-api:2.0.16")
}

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(17))
    }
}

lavalinkPlugin {
    name = "rappa-premium-plugin"
    apiVersion = "4.2.1"
    serverVersion = "4.2.2"
}
