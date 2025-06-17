#!/bin/node

/// <reference path="./types.d.ts" />
/// <reference path="./hook.d.ts" />

define(() => {
  const GiB = 1024 * 1024 * 1024
  /**
   * @param {core.Record} item
   */
  return function cb(item) {
    let g = $app.runInTransaction((tx) => {
      let num = item.getInt("num")
      // 售价: 2元/GiB, 所以 1 元只有 0.5G
      let g = num * 0.5 * GiB
      let user = tx.findRecordById("users", item.getString("user"))
      let b = user.getInt("remaining_bytes")
      b = b + g
      user.set("remaining_bytes", b)
      tx.save(user)
    })
  }
})
