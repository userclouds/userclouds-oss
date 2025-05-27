package com.plexanalytics;

import org.apache.flink.streaming.api.functions.windowing.WindowFunction;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.apache.logging.log4j.Level;

/**
 * Class for summing up the count of events for a type in a given window
 */
public class CounterAggregator implements WindowFunction<LogEvent, CounterRecord, Integer, TimeWindow> {
    // Central framework logger
	private static final Logger log = LogManager.getLogger(CounterAggregator.class);
	// Allows for greater logging detail from the app within getting extra info from the framework
	final static Level CUSTOM = Level.forName("CUSTOM", 1);
	// Configuration flag controlling extra logging
    private boolean debugMode;

    
    public CounterAggregator(boolean debugMode) {
        this.debugMode = debugMode;
    }

	/**
	 * apply() is invoked once for each window.
	 *
	 * @param eventType the type of the event being aggregated
	 * @param window meta data for the window
	 * @param input an iterable over log events that were assigned to the window
	 * @param out a collector to emit results from the function
	 */
	@Override
	public void apply(Integer eventType, TimeWindow window, Iterable<LogEvent> input, Collector<CounterRecord> out) {

		// count the events within the time window
		int cnt = 0;

		for (LogEvent r : input) {
			if (this.debugMode) {
				log.log(CUSTOM, "Processed " + r.toString());  
			}
			cnt++;
		}

		// emit a total count of the events of a given eventType
		out.collect(new CounterRecord(eventType, window.getStart(), window.getEnd(), cnt));
	}
}
