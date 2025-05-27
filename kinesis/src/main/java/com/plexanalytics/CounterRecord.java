package com.plexanalytics;

import java.util.Date; 

/**
 * Class for storing the aggregated counter information
 */
public class CounterRecord {

    // Event type
    private int type;
    // End of the window marker
    private long timestampStart;
    // End of the window marker
    private long timestampEnd;
    // Number of events that occured during the window
    private int count;

    public CounterRecord(int type, long timestampStart, long timestampEnd, int count) {
        this.type = type;
        this.timestampStart = timestampStart;
        this.timestampEnd = timestampEnd;
        this.count = count;
    }

    public int getType() {
		return type;
	}

	public long setWindowStart() {
		return this.timestampStart;
	}

    public long setWindowEnd() {
		return this.timestampEnd;
	}

	public int getCount() {
		return count;
	}

    public String toString() {
        Date dateS = new java.util.Date(this.timestampStart);
        Date dateE = new java.util.Date(this.timestampEnd);
        return "(" + this.type + ", " + dateS.toString() + " " + dateE.toString() + ", " + this.count + ")";
    }
}
