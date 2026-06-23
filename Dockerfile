# Use a lightweight Debian image for maximum compatibility with Go binaries (glibc)
FROM debian:bookworm-slim

# docker build -t mitm-aggregator:latest -f scheduler/mitm_scheduler/Dockerfile .
# docker run -d --name mitm-app \
#       -e SCHEDULER_PASSWORD="DeinPasswort123!" \
#       -e MASTER_KEY="<DeinBase64MasterKey>" \
#       -p 8080:8080 \
#       mitm-aggregator:latest


# Install CA certificates (required for SaaS adapters and HTTPS calls)
RUN apt-get update && \
    apt-get install -y apt-utils ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy all pre-compiled Go binaries from the local ./bin directory into the container
COPY ./bin/ /app/bin/

# Ensure all binaries are executable
RUN chmod +x /app/bin/*

# Add the /app/bin directory to the PATH so the scheduler can seamlessly spawn the collectors,
# transformer, and delivery binaries without needing absolute paths.
ENV PATH="/app/bin:${PATH}"

EXPOSE 8080
# The primary process for this container is the MitM Scheduler (mitm-server).
# It will spawn the other binaries in the background via os.Exec when jobs trigger.
ENTRYPOINT ["mitm-server"]

# Default command argument (points to the pre-encrypted config file copied into /app/bin/)
# Example: docker run -d -e SCHEDULER_PASSWORD=... my-mitm-image
CMD ["/app/bin/config.enc"]
