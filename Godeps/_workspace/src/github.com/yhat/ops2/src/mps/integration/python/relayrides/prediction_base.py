import cPickle as pickle
import pandas as pd

from relayrides.connection import MysqlConnection

price_query = '''SELECT max_rate.vehicle_id, daily, weekly, monthly, IF(primary_contact_id IN (111355,381453), 1, 0) as is_lot
                    FROM (
                        SELECT vehicle_id, MAX(id) as id
                        FROM relayrides.rate
                        GROUP BY vehicle_id) max_rate
                    JOIN relayrides.rate
                        ON rate.id = max_rate.id
                    JOIN relayrides.vehicle
                        ON vehicle.id = rate.vehicle_id
                    WHERE rate.vehicle_id NOT IN (SELECT search_index_event.vehicle_id
                        FROM (
                            SELECT search_index_event.vehicle_id, MAX(id) as id
                            FROM relayrides.search_index_event
                            GROUP BY vehicle_id
                            ) max_search
                        JOIN relayrides.search_index_event
                            ON search_index_event.id = max_search.id
                            WHERE exclusion_reason IS NOT NULL) AND rate.vehicle_id IN (SELECT vehicle_listing_enabled.vehicle_id
                        FROM (
                            SELECT vehicle_id, MAX(id) as id
                            FROM relayrides.vehicle_listing_enabled
                            GROUP BY vehicle_id
                            ) max_enabled
                        JOIN relayrides.vehicle_listing_enabled
                            ON vehicle_listing_enabled.id = max_enabled.id
                            WHERE enabled = 1
                            );'''

if __name__ == '__main__':
    c = MysqlConnection(ssh=True)
    price_df = c.fetch_data(price_query, 'df')
    price_df.set_index('vehicle_id', inplace = True)
    with open('price_df.pkl', 'wb') as f:
        pickle.dump(price_df, f)
