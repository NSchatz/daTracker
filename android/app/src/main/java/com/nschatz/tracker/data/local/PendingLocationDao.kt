package com.nschatz.tracker.data.local

import androidx.room.Dao
import androidx.room.Insert
import androidx.room.OnConflictStrategy
import androidx.room.Query

@Dao
interface PendingLocationDao {

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insert(entity: PendingLocationEntity)

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertAll(entities: List<PendingLocationEntity>)

    @Query("SELECT * FROM pending_locations ORDER BY id ASC LIMIT :limit")
    suspend fun getOldest(limit: Int): List<PendingLocationEntity>

    @Query("DELETE FROM pending_locations WHERE id IN (:ids)")
    suspend fun deleteByIds(ids: List<Long>)

    @Query("SELECT COUNT(*) FROM pending_locations")
    suspend fun count(): Int
}
