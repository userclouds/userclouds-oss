package com.plexanalytics;

import java.util.Date; 

public class LogEvent {

	private String content;
	private int type;
	private String applicationid;
	private long timestamp;

	public String getContent() {
		return content;
	}

	public void setContent(String content) {
		this.content = content;
	}

	public int getType() {
		return type;
	}

	public void setType(int type) {
		this.type = type;
	}

	public long getTimestamp() {
		return timestamp;
	}

	public void setTimestamp(long timestamp) {
		this.timestamp = timestamp;
	}

	public String getApplicationId() {
		return applicationid;
	}

	public void setApplicationId(String applicationid) {
		this.applicationid = applicationid;
	}
	public String toString() {
        Date date = new java.util.Date(this.timestamp * 1000);
 
        return "(" + this.type + ", " + date.toString() + ", " + this.content + ")";
    }
}
