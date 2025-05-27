This is the source code for a dbt-cloud project for demoing ease of tokenization of data in BigQuery via dbt.

To set up the demo, you can mostly follow the instructions here:
https://www.datafold.com/blog/dbt-python#how-to-build-dbt-python-models-in-bigquery

A few notes:

- make sure to use image: 2.1 (Ubuntu 20.04 LTS, Hadoop 3.3, Spark 3.3) (other versions don’t seem to work with spark-bigquery jar, regardless of version)
- make sure to use n1-standard-4 machine (n2 doesn’t seem to work with the above image, and n1-standard-2 doesn’t have enough memory)
- need to open up firewall for dbt servers
- the connectors.sh script should be modified to include “pip install usercloudssdk” at the end (or a separate script)
