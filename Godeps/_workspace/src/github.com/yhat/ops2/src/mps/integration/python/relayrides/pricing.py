from datetime import datetime, timedelta
from dateutil import relativedelta
import pandas as pd
import cPickle as pickle

class Pricing(object):
    def __init__(self):
        self.price_df = pickle.load(open('price_df.pkl','rb'))
        self.insurance_dict = {'decline': 0, 'basic': 0.15, 'premium': 0.4}

    def get_price(self, vehicle_id, start_ts, end_ts, insurance = 'decline'):
        try:
            [daily_rate, weekly_rate, monthly_rate] = self.price_df[['daily', 'weekly', 'monthly']].ix[vehicle_id].tolist()
        except:
            raise Exception('vehicle_id not in vehicle base')
        if start_ts >= end_ts:
            raise Exception('start_ts should be lower than end_ts')
        months_in_interval = relativedelta.relativedelta(end_ts, start_ts).months
        weeks_in_interval = relativedelta.relativedelta(end_ts, start_ts + relativedelta.relativedelta(months = months_in_interval)).days / 7
        days_in_interval = relativedelta.relativedelta(end_ts, start_ts + relativedelta.relativedelta(months = months_in_interval)).days % 7
        if start_ts + relativedelta.relativedelta(months = months_in_interval, days = 7 * weeks_in_interval + days_in_interval) < end_ts:
            if days_in_interval == 6:
                weeks_in_interval += 1
            else:
                days_in_interval += 1
        if days_in_interval * daily_rate > weekly_rate:
            weeks_in_interval += 1
            days_in_interval = 0
        if weeks_in_interval * weekly_rate > monthly_rate:
            months_in_interval += 1
            weeks_in_interval = 0
        boost = 0.1 + self.insurance_dict[insurance]
        if self.price_df.ix[vehicle_id]['is_lot']:
            boost += 0.1
        return round((1 + boost) * (daily_rate * days_in_interval + weeks_in_interval * weekly_rate + months_in_interval * monthly_rate), 2)

    def predict(self, data):
        vehicle_id = int(data['vehicle_id'])
        start_date = data['start_date']
        start_time = data['start_time']
        start_ts = datetime.strptime(start_date + ' ' + start_time, '%m/%d/%Y %H:%M')
        end_date = data['end_date']
        end_time = data['end_time']
        end_ts = datetime.strptime(end_date + ' ' + end_time, '%m/%d/%Y %H:%M')
        if 'insurance' in data:
            insurance = data['insurance']
        else:
            insurance = 'decline'
        return self.get_price(vehicle_id, start_ts, end_ts, insurance)
