package com.nschatz.tracker.data.local

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase

@Database(
    entities = [PendingLocationEntity::class, CachedLocationEntity::class],
    version = 1,
    exportSchema = false
)
abstract class TrackerDatabase : RoomDatabase() {

    abstract fun pendingLocationDao(): PendingLocationDao
    abstract fun cachedLocationDao(): CachedLocationDao

    companion object {
        @Volatile
        private var INSTANCE: TrackerDatabase? = null

        fun getInstance(context: Context): TrackerDatabase {
            return INSTANCE ?: synchronized(this) {
                INSTANCE ?: Room.databaseBuilder(
                    context.applicationContext,
                    TrackerDatabase::class.java,
                    "tracker.db"
                ).build().also { INSTANCE = it }
            }
        }
    }
}
