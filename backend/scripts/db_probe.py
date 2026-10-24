# -*- coding: utf-8 -*-
""" secretive: namer.db 写入探针（只用一次）"""
import sqlite3
import sys

con = sqlite3.connect(r"D:\19-Training\learngo\name\backend\namer.db", timeout=3)
try:
    print("journal_mode:", con.execute("pragma journal_mode").fetchone())
    con.execute("insert or ignore into favorites(id,surname,given_name,pinyin,gender,score,source,notes,created_at) values('probe-x','t','t','','male',50,'probe','',datetime('now'))")
    con.commit()
    print("WRITE_OK")
    con.execute("delete from favorites where id='probe-x'")
    con.commit()
except Exception as e:  # noqa: BLE001
    print("WRITE_ERR:", type(e).__name__, e)
finally:
    con.close()
