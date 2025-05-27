package com.plexanalytics;

import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.Properties;
import java.util.concurrent.TimeUnit;

import org.apache.flink.api.common.serialization.SimpleStringEncoder;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.api.java.utils.ParameterTool;
import org.apache.flink.core.fs.Path;
import org.apache.flink.shaded.jackson2.com.fasterxml.jackson.databind.ObjectMapper;
import org.apache.flink.streaming.api.TimeCharacteristic;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.datastream.KeyedStream;
import org.apache.flink.streaming.api.environment.LocalStreamEnvironment;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.datastream.SingleOutputStreamOperator;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.apache.flink.streaming.api.functions.sink.filesystem.StreamingFileSink;
import org.apache.flink.streaming.api.functions.sink.filesystem.rollingpolicies.DefaultRollingPolicy;
import org.apache.flink.streaming.api.windowing.assigners.ProcessingTimeSessionWindows;
import org.apache.flink.streaming.api.windowing.time.Time;
import org.apache.flink.streaming.connectors.kinesis.FlinkKinesisConsumer;
import org.apache.flink.streaming.connectors.kinesis.config.ConsumerConfigConstants;
import org.apache.flink.util.OutputTag;
import org.apache.flink.util.Collector;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.apache.logging.log4j.Level;

import com.amazonaws.services.kinesisanalytics.runtime.KinesisAnalyticsRuntime;

public class SessionProcessor {

	private static final Logger log = LogManager.getLogger(SessionProcessor.class);
	// Allows for greater logging detail from the app within getting extra info from the framework
	final static Level CUSTOM = Level.forName("CUSTOM", 1);

	public enum StreamPosition {
		LATEST,
		TRIM_HORIZON,
		AT_TIMESTAMP
	}

	/**
	 * Main method and the entry point for Kinesis Data Analytics Flink Application.
	 *
	 * @param args
	 * @throws Exception
	 */
	public static void main(String[] args) throws Exception {
		StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Error out if being run locallu
		if (env instanceof LocalStreamEnvironment) {
			throw new RuntimeException(
					"Code is not configured to run locally. Exiting");

		}
		// Read properties from Kinesis Data Analytics environment
		Map<String, Properties> applicationProperties = KinesisAnalyticsRuntime.getApplicationProperties();
		Properties flinkProperties = applicationProperties.get("PlexAnalyticsProperties");
		if (flinkProperties == null) {
			throw new RuntimeException("Unable to load properties from Group ID PlexAnalyticsProperties.");
		}
		ParameterTool parameter = SessionProcessor.fromApplicationProperties(flinkProperties);

		if (!validateRuntimeRemoteProperties(parameter))
			throw new RuntimeException(
				"Runtime properties are invalid. Exiting");

		// Figure out if extra logging is enabled
		boolean debugMode = Boolean.parseBoolean(parameter.get("debug_mode"));

		// Label the events depending on the time they were received to prevent reordering them due to drift
		env.setStreamTimeCharacteristic(TimeCharacteristic.IngestionTime);
		env.registerType(LogEvent.class);

		// Connect to the input kinesis stream
		DataStream<String> stream = createKinesisSource(env, parameter);
		log.log(CUSTOM, "Kinesis stream created.");

		// Parse the incoming stream dropping all messages that can't be parsed
		ObjectMapper objectMapper = new ObjectMapper();
		DataStream<LogEvent> parsedStream = stream.map(record -> {
			try {
				if (debugMode) {
					log.log(CUSTOM, "Parsing " + record);
				}
				return objectMapper.readValue(record, LogEvent.class);
			} catch (Exception e) {
				if (debugMode) {
					log.log(CUSTOM,"Parsing error. Input record: " + record);
				}
				return null;
			}
		}).filter(Objects::nonNull);

		// Split the stream into messages which will go to file storage without processing and counters which will be aggregated
        final OutputTag<LogEvent> outputTagMessages = new OutputTag<LogEvent>("messages"){};

        SingleOutputStreamOperator<LogEvent> splitStream = parsedStream
          .process(new ProcessFunction<LogEvent, LogEvent>() {
              @Override
              public void processElement(
                  LogEvent record,
                  Context ctx,
                  Collector<LogEvent> out) throws Exception {
                    if (record.getType() > 15) { // TODO this value is shared acrodd GO and Java
                        // emit data to main stream for counters if it is a countable event
                        out.collect(record);
                    } else {
						// emit data to the side stream for the S3 storage which contains all info/debug/etc
						// events
                        ctx.output(outputTagMessages, record);
                    }
              }
            });

		// Split the counter stream by event type
		KeyedStream<LogEvent, Integer> keyedStream = splitStream.keyBy(LogEvent::getType);
		// Break up each per type stream into a window of "timeout" minutes within which the
		// events will be counted
		long timeout = Long.parseLong(parameter.get("session_time_out_in_minutes"));
		DataStream<String> sessionStream = keyedStream
				.window(ProcessingTimeSessionWindows.withGap(Time.minutes(timeout)))
				.apply(new CounterAggregator(debugMode))
				.map(r -> r.toString())
				.name("session_stream");

		sessionStream.addSink(createS3Sink(parameter, "counters")).name("counter_sink");
		log.log(CUSTOM, "Counters S3 Sink added.");

		DataStream<String> messageStream = splitStream.getSideOutput(outputTagMessages).map(r -> r.toString());
        messageStream.addSink(createS3Sink(parameter, "messages")).name("message_sink");
		log.log(CUSTOM, "Messages S3 Sink added.");

		env.execute("Plex Basic Processing");
	}

	private static ParameterTool fromApplicationProperties(Properties properties) {
		Map<String, String> map = new HashMap<>(properties.size());
		properties.forEach((k, v) -> map.put((String) k, (String) v));
		return ParameterTool.fromMap(map);
	}

	/**
	 * Method creates Kinesis source based on Application properties
	 *
	 * @param env
	 * @param parameter
	 * @return
	 */
	private static DataStream<String> createKinesisSource(StreamExecutionEnvironment env, ParameterTool paramTool) {
		log.info("Creating Kinesis source from Application Properties");
		Properties inputProperties = new Properties();
		inputProperties.setProperty(ConsumerConfigConstants.AWS_REGION, paramTool.get("region"));
		inputProperties.setProperty(ConsumerConfigConstants.STREAM_INITIAL_POSITION,
				paramTool.get("stream_init_position"));
		if (paramTool.get("stream_init_position").equalsIgnoreCase(StreamPosition.AT_TIMESTAMP.name())) {
			inputProperties.setProperty(ConsumerConfigConstants.STREAM_INITIAL_TIMESTAMP,
					paramTool.get("stream_initial_timestamp"));
		}

		String inputName = paramTool.get("input_stream_name");
		return env.addSource(new FlinkKinesisConsumer<>(inputName, new SimpleStringSchema(),
				inputProperties));
	}

	/**
	 * Method creates S3 sink based on application properties
	 *
	 * @param parameter
	 * @return
	 */
	private static StreamingFileSink<String> createS3Sink(ParameterTool paramTool, String subfolder) {
		String outputName = paramTool.get("s3_output_path") + "/" + subfolder;
		log.log(CUSTOM, "Creating S3 sink from Application Properties - " + outputName);
		final StreamingFileSink<String> sink = StreamingFileSink
				.forRowFormat(new Path(paramTool.get("s3_output_path") + "/" + subfolder), new SimpleStringEncoder<String>("UTF-8"))
				.withBucketCheckInterval(
						TimeUnit.SECONDS.toMillis(Long.parseLong(paramTool.get("bucket_check_interval_in_seconds"))))
				.withRollingPolicy(DefaultRollingPolicy.create()
						.withRolloverInterval(
								TimeUnit.SECONDS.toMillis(Long.parseLong(paramTool.get("rolling_interval_in_seconds"))))
						.withInactivityInterval(TimeUnit.SECONDS
								.toMillis(Long.parseLong(paramTool.get("inactivity_interval_in_seconds"))))
						.build())
				.build();
		return sink;
	}

	/**
	 * Method validates runtime properties
	 *
	 * @param parameter
	 * @return
	 */
	private static boolean validateRuntimeRemoteProperties(ParameterTool paramTool) {

		boolean bucketExist = false;
		boolean streamExist = false;
		boolean propertiesValid = false;
		boolean initialTimestampNAOrValidIfPresent = false;
		long sessionTimeout = 0;


		// Log the incoming configuration parameters for debugging
		log.log(CUSTOM, "Printing runtime Properties to CloudWatch");
		paramTool.toMap().forEach((key, value) -> log.log(CUSTOM, "parameter: " + key + ", value: " + value));
		// Check if the output S3 bucket is valid and the app has access rights
		bucketExist = SessionUtil.checkIfBucketExist(paramTool.get("region"), paramTool.get("s3_output_path"));
		// Check if the input kinesis stream exists and the app has access rights
		streamExist = SessionUtil.checkIfStreamExist(paramTool.get("region"), paramTool.get("input_stream_name"));
		try {
			sessionTimeout = Long.parseLong(paramTool.get("session_time_out_in_minutes"));
		} catch (NumberFormatException e) {
			log.error("Value for property 'session_time_out_in_minutes' is invalid");
		}

		// Check if stream_init_position is valid
		boolean streamInitPositionValid = Arrays.stream(StreamPosition.values())
				.anyMatch((t) -> t.name().equals(paramTool.get("stream_init_position")));
        // Validate the timestamp if required by the stream position selection
		if (streamInitPositionValid) {
			if (paramTool.get("stream_init_position").equalsIgnoreCase(StreamPosition.AT_TIMESTAMP.name())) {
				if (Optional.ofNullable(paramTool.get("stream_initial_timestamp")).isPresent()) {
					if (SessionUtil.validateDate(paramTool.get("stream_initial_timestamp")))
						initialTimestampNAOrValidIfPresent = true;
				} else
					log.error(
						"stream_init_position is set to 'AT_TIMESTAMP' but 'stream_initial_timestamp' is not provided");
			} else
				initialTimestampNAOrValidIfPresent = true;
		}
		// Check if all conditions are met
		if (sessionTimeout != 0L && streamExist && bucketExist && streamInitPositionValid
				&& initialTimestampNAOrValidIfPresent) {
			propertiesValid = true;
			log.log(CUSTOM, "Runtime properties are valid.");
		} else {
			log.error("Runtime properties are not valid.");
			if (!streamExist)
				log.error("The specified Kinesis stream: " + paramTool.get("input_stream_name") + "does not exist.");
			if (!bucketExist)
				log.error("The specified s3 bucket: " + paramTool.get("s3_output_path") + "does not exist.");
		}

		// Log details about how the input data will be read
		if (propertiesValid) {
			log.log(CUSTOM, "Wll consume data from: " + paramTool.get("stream_init_position"));
			if (paramTool.get("stream_init_position").equalsIgnoreCase(StreamPosition.AT_TIMESTAMP.name())) {
				log.log(CUSTOM, "The 'STREAM_INITIAL_TIMESTAMP' is set to: " + paramTool.get("stream_initial_timestamp"));
			}

		}
		return propertiesValid;
	}
}
