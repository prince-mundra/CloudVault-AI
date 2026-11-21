from psycopg.rows import dict_row
from psycopg_pool import AsyncConnectionPool

from ai_service.config import Settings


def create_pool(settings: Settings) -> AsyncConnectionPool:
    return AsyncConnectionPool(
        conninfo=settings.database_url,
        min_size=1,
        max_size=8,
        open=False,
        kwargs={"row_factory": dict_row},
    )
