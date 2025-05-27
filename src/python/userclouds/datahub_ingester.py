#! /usr/bin/env python3


import asyncio
import logging
import sys

import datahub as datahub_package
from datahub.cli.ingest_cli import logger
from datahub.configuration.config_loader import load_config_file
from datahub.ingestion.run.pipeline import Pipeline
from datahub.upgrade import upgrade


# based on https://github.com/datahub-project/datahub/blob/master/metadata-ingestion/src/datahub/cli/ingest_cli.py
def dh_ingest():
    logging.basicConfig(level=logging.DEBUG)
    logger.info("DataHub CLI version: %s", datahub_package.nice_version_name())
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} <config_file>")
        exit(1)
    config_file = sys.argv[1]
    print(f"Using config file: {config_file}")
    pipeline_config = load_config_file(
        config_file,
        squirrel_original_config=True,
        squirrel_field="__raw_config",
        allow_stdin=True,
        allow_remote=True,
        process_directives=True,
        resolve_env_vars=True,
    )
    loop = asyncio.get_event_loop()
    ret = loop.run_until_complete(run_ingestion_and_check_upgrade(pipeline_config))
    if ret:
        sys.exit(ret)


async def run_pipeline_to_completion(pipeline: Pipeline) -> int:
    logger.info("Starting metadata ingestion")
    try:
        pipeline.run()
    except Exception as e:
        logger.info(
            f"Source ({pipeline.config.source.type}) report:\n{pipeline.source.get_report().as_string()}"
        )
        logger.info(
            f"Sink ({pipeline.config.sink.type}) report:\n{pipeline.sink.get_report().as_string()}"
        )
        raise e
    else:
        logger.info("Finished metadata ingestion")
        pipeline.log_ingestion_stats()
        ret = pipeline.pretty_print_summary(warnings_as_failure=False)
        return ret


async def run_ingestion_and_check_upgrade(pipeline_config) -> int:
    # TRICKY: We want to make sure that the Pipeline.create() call happens on the
    # same thread as the rest of the ingestion. As such, we must initialize the
    # pipeline inside the async function so that it happens on the same event
    # loop, and hence the same thread.

    # logger.debug(f"Using config: {pipeline_config}")
    raw_pipeline_config = pipeline_config.pop("__raw_config")
    pipeline = Pipeline.create(
        pipeline_config, dry_run=False, raw_config=raw_pipeline_config
    )

    version_stats_future = asyncio.ensure_future(
        upgrade.retrieve_version_stats(pipeline.ctx.graph)
    )
    ingestion_future = asyncio.ensure_future(run_pipeline_to_completion(pipeline))
    ret = await ingestion_future

    # The main ingestion has completed. If it was successful, potentially show an upgrade nudge message.
    if ret == 0:
        try:
            # we check the other futures quickly on success
            version_stats = await asyncio.wait_for(version_stats_future, 0.5)
            upgrade.maybe_print_upgrade_message(version_stats=version_stats)
        except Exception as e:
            logger.debug(
                f"timed out with {e} waiting for version stats to be computed... skipping ahead."
            )
    return ret


if __name__ == "__main__":
    dh_ingest()
