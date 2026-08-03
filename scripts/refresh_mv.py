"""
refresh_mv.py
=============
1) 创建 stock_history_mv 物化视图（如果不存在）
2) 增量刷新（如已存在，带 CONCURRENTLY）
运行: python refresh_mv.py
"""
import configparser
import os
import sys
from sqlalchemy import create_engine, text
from pathlib import Path

DDL_PATH = Path(__file__).parent / "refresh_mv.sql"


def main():
    cfg_path = os.environ.get("CONFIG_INI", "config.ini")
    config = configparser.ConfigParser()
    config.read(cfg_path, encoding="utf-8")
    db_url = config.get("database", "url")

    engine = create_engine(db_url)
    ddl = DDL_PATH.read_text(encoding="utf-8")

    with engine.begin() as conn:
        exists = conn.execute(text("""
            SELECT 1 FROM pg_matviews WHERE matviewname = 'stock_history_mv'
        """)).first()

        if not exists:
            print("→ 首次创建 stock_history_mv ...")
            conn.execute(text(ddl))
            print("✅ 创建完成")
        else:
            print("→ MV 已存在，CONCURRENTLY 刷新 ...")
            conn.execute(text("REFRESH MATERIALIZED VIEW CONCURRENTLY stock_history_mv;"))
            print("✅ 增量刷新完成")

        cnt = conn.execute(text("SELECT COUNT(*) FROM stock_history_mv")).scalar()
        print(f"   MV 当前行数: {cnt}")


if __name__ == "__main__":
    main()
