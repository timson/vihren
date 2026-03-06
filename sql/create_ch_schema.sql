-- create database
CREATE DATABASE IF NOT EXISTS flamedb;

CREATE TABLE IF NOT EXISTS flamedb.samples_in
(
    Timestamp        DateTime('UTC'),
    Service          LowCardinality(String),
    InstanceType     LowCardinality(String),
    ContainerEnvName LowCardinality(String),
    HostName         LowCardinality(String),
    ContainerName    LowCardinality(String),
    NumSamples       UInt32,
    CallStackHash    Int64,
    CallStackName    String,
    CallStackParent  Int64
    )
    ENGINE = Null();

-- raw samples (local)
CREATE TABLE IF NOT EXISTS flamedb.samples
(
    Timestamp        DateTime('UTC'),
    Service          LowCardinality(String),
    InstanceType     LowCardinality(String),
    ContainerEnvName LowCardinality(String),
    HostName         LowCardinality(String),
    ContainerName    LowCardinality(String),
    NumSamples       UInt32,
    CallStackHash    Int64,
    CallStackParent  Int64
    )
    ENGINE = MergeTree
    PARTITION BY toYYYYMMDD(Timestamp)
    ORDER BY (Service, InstanceType, ContainerEnvName, HostName, Timestamp)
    TTL Timestamp + INTERVAL 30 DAY;


CREATE MATERIALIZED VIEW IF NOT EXISTS flamedb.samples_mv
TO flamedb.samples
AS
SELECT
    Timestamp,
    Service,
    InstanceType,
    ContainerEnvName,
    HostName,
    ContainerName,
    NumSamples,
    CallStackHash,
    CallStackParent
FROM flamedb.samples_in;


-- callstack names
CREATE TABLE IF NOT EXISTS flamedb.samples_name
(
    Timestamp DateTime('UTC'),
    Service LowCardinality(String),
    CallStackHash Int64,
    CallStackName String CODEC(ZSTD)
)
    ENGINE = ReplacingMergeTree
    PARTITION BY toYYYYMMDD(Timestamp)
    ORDER BY (Timestamp, Service, CallStackHash)
    TTL Timestamp + INTERVAL 30 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS flamedb.samples_name_mv
TO flamedb.samples_name
AS
SELECT
    toStartOfDay(Timestamp) AS Timestamp,
    Service,
    CallStackHash,
    any(CallStackName)   AS CallStackName
FROM flamedb.samples_in
GROUP BY
    Service,
    CallStackHash,
    Timestamp;


-- raw metrics (local)
CREATE TABLE IF NOT EXISTS flamedb.metrics
(
    Timestamp                DateTime('UTC'),
    Service                  LowCardinality(String),
    InstanceType             LowCardinality(String),
    HostName                 LowCardinality(String),
    CPUAverageUsedPercent    Float64,
    MemoryAverageUsedPercent Float64
    )
    ENGINE = MergeTree
    PARTITION BY toYYYYMMDD(Timestamp)
    ORDER BY (Service, InstanceType, HostName, Timestamp);

-- 1 hour aggregated (all hostnames, all containers)
CREATE TABLE IF NOT EXISTS flamedb.samples_1hour_all
(
    Timestamp       DateTime('UTC'),
    Service         LowCardinality(String),
    CallStackHash   Int64,
    CallStackParent Int64,
    NumSamples      Int64
    )
    ENGINE = SummingMergeTree(NumSamples)
    PARTITION BY toYYYYMMDD(Timestamp)
    ORDER BY (Service, Timestamp, CallStackHash, CallStackParent)
    TTL Timestamp + INTERVAL 30 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS flamedb.samples_1hour_all_mv
TO flamedb.samples_1hour_all
AS
SELECT
    toStartOfHour(Timestamp) AS Timestamp,
    Service,
    CallStackHash,
    CallStackParent,
    sum(NumSamples)      AS NumSamples
FROM flamedb.samples
GROUP BY
    Service,
    CallStackHash,
    CallStackParent,
    Timestamp;

-- 1 hour aggregated (by instance/host/container)
CREATE TABLE IF NOT EXISTS flamedb.samples_1hour
(
    Timestamp        DateTime('UTC'),
    Service          LowCardinality(String),
    InstanceType     LowCardinality(String),
    ContainerEnvName LowCardinality(String),
    HostName         LowCardinality(String),
    ContainerName    LowCardinality(String),
    CallStackHash    Int64,
    CallStackParent  Int64,
    NumSamples       Int64
    )
    ENGINE = SummingMergeTree(NumSamples)
    PARTITION BY toYYYYMMDD(Timestamp)
    ORDER BY (
                 Service,
                 ContainerEnvName,
                 InstanceType,
                 HostName,
                 ContainerName,
                 Timestamp,
                 CallStackHash,
                 CallStackParent
             )
    TTL Timestamp + INTERVAL 30 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS flamedb.samples_1hour_mv
TO flamedb.samples_1hour
AS
SELECT
    toStartOfHour(Timestamp) AS Timestamp,
    Service,
    ContainerEnvName,
    InstanceType,
    HostName,
    ContainerName,
    CallStackHash,
    CallStackParent,
    sum(NumSamples)      AS NumSamples
FROM flamedb.samples
GROUP BY
    Service,
    InstanceType,
    ContainerEnvName,
    HostName,
    ContainerName,
    CallStackHash,
    CallStackParent,
    Timestamp;

-- 1 day aggregated (all hostnames, all containers)
CREATE TABLE IF NOT EXISTS flamedb.samples_1day_all
(
    Timestamp       DateTime('UTC'),
    Service         LowCardinality(String),
    CallStackHash   Int64,
    CallStackParent Int64,
    NumSamples      Int64
    )
    ENGINE = SummingMergeTree(NumSamples)
    PARTITION BY toYYYYMMDD(Timestamp)
    ORDER BY (Service, Timestamp, CallStackHash, CallStackParent)
    TTL Timestamp + INTERVAL 30 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS flamedb.samples_1day_all_mv
TO flamedb.samples_1day_all
AS
SELECT
    toStartOfDay(Timestamp) AS Timestamp,
    Service,
    CallStackHash,
    CallStackParent,
    sum(NumSamples)      AS NumSamples
FROM flamedb.samples
GROUP BY
    Service,
    CallStackHash,
    CallStackParent,
    Timestamp;

-- 1 day aggregated (by instance/host/container)
CREATE TABLE IF NOT EXISTS flamedb.samples_1day
(
    Timestamp        DateTime('UTC'),
    Service          LowCardinality(String),
    InstanceType     LowCardinality(String),
    ContainerEnvName LowCardinality(String),
    HostName         LowCardinality(String),
    ContainerName    LowCardinality(String),
    CallStackHash    Int64,
    CallStackParent  Int64,
    NumSamples       Int64
    )
    ENGINE = SummingMergeTree(NumSamples)
    PARTITION BY toYYYYMMDD(Timestamp)
    ORDER BY (
                 Service,
                 ContainerEnvName,
                 InstanceType,
                 HostName,
                 ContainerName,
                 Timestamp,
                 CallStackHash,
                 CallStackParent
             )
    TTL Timestamp + INTERVAL 365 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS flamedb.samples_1day_mv
TO flamedb.samples_1day
AS
SELECT
    toStartOfDay(Timestamp) AS Timestamp,
    Service,
    ContainerEnvName,
    InstanceType,
    HostName,
    ContainerName,
    CallStackHash,
    CallStackParent,
    sum(NumSamples)      AS NumSamples
FROM flamedb.samples
GROUP BY
    Service,
    InstanceType,
    ContainerEnvName,
    HostName,
    ContainerName,
    CallStackHash,
    CallStackParent,
    Timestamp;

-- 1 minute aggregated (roots only)
CREATE TABLE IF NOT EXISTS flamedb.samples_1min
(
    Timestamp        DateTime('UTC'),
    Service          LowCardinality(String),
    InstanceType     LowCardinality(String),
    ContainerEnvName LowCardinality(String),
    HostName         LowCardinality(String),
    ContainerName    LowCardinality(String),
    NumSamples       Int64
    )
    ENGINE = SummingMergeTree(NumSamples)
    PARTITION BY toYYYYMMDD(Timestamp)
    ORDER BY (Service, ContainerEnvName, InstanceType, ContainerName, HostName, Timestamp);

CREATE MATERIALIZED VIEW IF NOT EXISTS flamedb.samples_1min_mv
TO flamedb.samples_1min
AS
SELECT
    toStartOfMinute(Timestamp) AS Timestamp,
    Service,
    InstanceType,
    ContainerEnvName,
    HostName,
    ContainerName,
    sum(NumSamples) AS NumSamples
FROM flamedb.samples
WHERE CallStackParent = 0
GROUP BY
    Service,
    InstanceType,
    ContainerEnvName,
    HostName,
    ContainerName,
    Timestamp;

