-- Seed: sentiment_lexicon_seed.sql
-- Purpose: Populate sentiment lexicon with WSB/Reddit financial slang
-- Usage: Run after migration 020_create_social_sentiment_tables.sql
-- Date: 2025-11-30

-- Bullish terms
INSERT INTO sentiment_lexicon (term, sentiment, weight, category) VALUES
-- Slang
('to the moon', 'bullish', 1.50, 'slang'),
('diamond hands', 'bullish', 1.20, 'slang'),
('tendies', 'bullish', 1.00, 'slang'),
('buy the dip', 'bullish', 1.30, 'slang'),
('rocket', 'bullish', 1.20, 'slang'),
('moon', 'bullish', 1.00, 'slang'),
('YOLO', 'bullish', 1.40, 'slang'),
('squeeze', 'bullish', 1.30, 'slang'),
('gamma squeeze', 'bullish', 1.50, 'slang'),
('short squeeze', 'bullish', 1.50, 'slang'),
('apes together strong', 'bullish', 1.30, 'slang'),
('hold the line', 'bullish', 1.20, 'slang'),
('ape', 'bullish', 0.90, 'slang'),
('apes', 'bullish', 0.90, 'slang'),
('load up', 'bullish', 1.20, 'slang'),
('free money', 'bullish', 1.30, 'slang'),
('cant go tits up', 'bullish', 1.40, 'slang'),
('literally cannot go tits up', 'bullish', 1.50, 'slang'),
('mooning', 'bullish', 1.30, 'slang'),
('stonks', 'bullish', 0.80, 'slang'),
('hodl', 'bullish', 1.10, 'slang'),
('btfd', 'bullish', 1.30, 'slang'),
('lambo', 'bullish', 1.20, 'slang'),
('when lambo', 'bullish', 1.10, 'slang'),
('wife boyfriend', 'bullish', 0.90, 'slang'),
('wifes boyfriend', 'bullish', 0.90, 'slang'),
('gain porn', 'bullish', 1.10, 'slang'),
('printing', 'bullish', 1.20, 'slang'),
('money printer', 'bullish', 1.10, 'slang'),
('brrr', 'bullish', 1.00, 'slang'),
('LFG', 'bullish', 1.30, 'slang'),
('lets go', 'bullish', 1.00, 'slang'),
('wagmi', 'bullish', 1.20, 'slang'),
('we like the stock', 'bullish', 1.30, 'slang'),
('send it', 'bullish', 1.20, 'slang'),
('all in', 'bullish', 1.30, 'slang'),
('balls deep', 'bullish', 1.30, 'slang'),
('degen', 'bullish', 0.80, 'slang'),
('degenerate', 'bullish', 0.80, 'slang'),
('hopium', 'bullish', 0.90, 'slang'),
('gigachad', 'bullish', 1.10, 'slang'),
('chad', 'bullish', 0.90, 'slang'),
('based', 'bullish', 0.90, 'slang'),
('massive gains', 'bullish', 1.40, 'slang'),
('huge gains', 'bullish', 1.30, 'slang'),
('moon mission', 'bullish', 1.40, 'slang'),
('rocket ship', 'bullish', 1.30, 'slang'),

-- Options terminology
('calls', 'bullish', 1.00, 'options'),
('call options', 'bullish', 1.00, 'options'),
('buying calls', 'bullish', 1.20, 'options'),
('deep itm', 'bullish', 1.10, 'options'),
('leaps', 'bullish', 1.00, 'options'),

-- Action words
('buy', 'bullish', 0.80, 'action'),
('buying', 'bullish', 0.90, 'action'),
('bought', 'bullish', 0.80, 'action'),
('accumulate', 'bullish', 1.00, 'action'),
('accumulating', 'bullish', 1.00, 'action'),
('adding', 'bullish', 0.90, 'action'),
('loading', 'bullish', 1.00, 'action'),
('loaded', 'bullish', 1.00, 'action'),
('averaging down', 'bullish', 0.90, 'action'),
('DCA', 'bullish', 0.90, 'action'),
('dollar cost averaging', 'bullish', 0.90, 'action'),

-- Position terms
('long', 'bullish', 1.00, 'position'),
('going long', 'bullish', 1.10, 'position'),

-- Analysis terms
('undervalued', 'bullish', 1.20, 'analysis'),
('oversold', 'bullish', 1.10, 'analysis'),
('breakout', 'bullish', 1.20, 'analysis'),
('breaking out', 'bullish', 1.20, 'analysis'),
('support', 'bullish', 0.70, 'analysis'),
('upside', 'bullish', 1.00, 'analysis'),
('catalyst', 'bullish', 1.00, 'analysis'),
('bullish divergence', 'bullish', 1.30, 'analysis'),
('bear trap', 'bullish', 1.30, 'analysis'),
('ATH', 'bullish', 1.10, 'analysis'),
('all time high', 'bullish', 1.10, 'analysis'),
('new highs', 'bullish', 1.10, 'analysis'),
('floor', 'bullish', 0.90, 'analysis'),
('higher lows', 'bullish', 1.00, 'analysis'),
('higher highs', 'bullish', 1.00, 'analysis'),
('gap up', 'bullish', 1.10, 'analysis'),
('ripping', 'bullish', 1.20, 'analysis'),

-- Direct sentiment
('bullish', 'bullish', 1.00, 'direct'),
('bull', 'bullish', 0.90, 'direct'),
('bulls', 'bullish', 0.90, 'direct'),

-- Emoji
('🚀', 'bullish', 1.50, 'emoji'),
('💎', 'bullish', 1.30, 'emoji'),
('🙌', 'bullish', 1.10, 'emoji'),
('🌙', 'bullish', 1.20, 'emoji'),
('🦍', 'bullish', 1.00, 'emoji'),
('💰', 'bullish', 1.00, 'emoji'),
('📈', 'bullish', 1.20, 'emoji'),
('🤑', 'bullish', 1.10, 'emoji'),
('🔥', 'bullish', 0.90, 'emoji'),
('💪', 'bullish', 0.80, 'emoji'),
('🐂', 'bullish', 1.00, 'emoji'),
('🦬', 'bullish', 1.00, 'emoji'),
('💹', 'bullish', 1.00, 'emoji'),
('🟢', 'bullish', 0.90, 'emoji'),
('⬆️', 'bullish', 0.80, 'emoji'),
('✅', 'bullish', 0.80, 'emoji'),
('🏆', 'bullish', 0.90, 'emoji'),
('👑', 'bullish', 0.90, 'emoji')

ON CONFLICT (term) DO NOTHING;

-- Bearish terms
INSERT INTO sentiment_lexicon (term, sentiment, weight, category) VALUES
-- Options terminology
('puts', 'bearish', 1.00, 'options'),
('put options', 'bearish', 1.00, 'options'),
('buying puts', 'bearish', 1.20, 'options'),
('deep otm puts', 'bearish', 1.30, 'options'),

-- Slang
('bagholders', 'bearish', 1.20, 'slang'),
('bagholder', 'bearish', 1.20, 'slang'),
('bagholding', 'bearish', 1.20, 'slang'),
('bag holding', 'bearish', 1.20, 'slang'),
('GUH', 'bearish', 1.50, 'slang'),
('paper hands', 'bearish', 1.10, 'slang'),
('paperhands', 'bearish', 1.10, 'slang'),
('dump', 'bearish', 1.20, 'slang'),
('dumping', 'bearish', 1.20, 'slang'),
('crash', 'bearish', 1.30, 'slang'),
('crashing', 'bearish', 1.30, 'slang'),
('loss porn', 'bearish', 1.00, 'slang'),
('rekt', 'bearish', 1.30, 'slang'),
('rug pull', 'bearish', 1.50, 'slang'),
('rugpull', 'bearish', 1.50, 'slang'),
('rugged', 'bearish', 1.40, 'slang'),
('drilling', 'bearish', 1.20, 'slang'),
('tanking', 'bearish', 1.20, 'slang'),
('bleeding', 'bearish', 1.10, 'slang'),
('worthless', 'bearish', 1.30, 'slang'),
('dead cat bounce', 'bearish', 1.20, 'slang'),
('fomo', 'bearish', 0.80, 'slang'),
('pump and dump', 'bearish', 1.40, 'slang'),
('scam', 'bearish', 1.40, 'slang'),
('fraud', 'bearish', 1.50, 'slang'),
('margin call', 'bearish', 1.30, 'slang'),
('margin called', 'bearish', 1.40, 'slang'),
('liquidated', 'bearish', 1.40, 'slang'),
('blown up', 'bearish', 1.30, 'slang'),
('FUD', 'bearish', 1.20, 'slang'),
('ngmi', 'bearish', 1.20, 'slang'),
('copium', 'bearish', 1.00, 'slang'),
('clapped', 'bearish', 1.20, 'slang'),
('blown account', 'bearish', 1.40, 'slang'),
('getting wrecked', 'bearish', 1.30, 'slang'),
('got wrecked', 'bearish', 1.30, 'slang'),
('wiped out', 'bearish', 1.30, 'slang'),
('capitulation', 'bearish', 1.30, 'slang'),
('capitulate', 'bearish', 1.20, 'slang'),
('blood in the streets', 'bearish', 1.40, 'slang'),
('falling knife', 'bearish', 1.30, 'slang'),
('catching a falling knife', 'bearish', 1.30, 'slang'),
('knife catching', 'bearish', 1.20, 'slang'),
('trap', 'bearish', 0.90, 'slang'),

-- Action words
('sell', 'bearish', 0.80, 'action'),
('selling', 'bearish', 0.90, 'action'),
('sold', 'bearish', 0.80, 'action'),
('exit', 'bearish', 0.70, 'action'),
('exiting', 'bearish', 0.80, 'action'),
('bail', 'bearish', 1.00, 'action'),
('bailing', 'bearish', 1.00, 'action'),

-- Position terms
('short', 'bearish', 1.00, 'position'),
('shorting', 'bearish', 1.10, 'position'),
('going short', 'bearish', 1.10, 'position'),

-- Analysis terms
('overvalued', 'bearish', 1.20, 'analysis'),
('overbought', 'bearish', 1.10, 'analysis'),
('breakdown', 'bearish', 1.20, 'analysis'),
('breaking down', 'bearish', 1.20, 'analysis'),
('resistance', 'bearish', 0.70, 'analysis'),
('downside', 'bearish', 1.00, 'analysis'),
('red flag', 'bearish', 1.20, 'analysis'),
('red flags', 'bearish', 1.20, 'analysis'),
('bearish divergence', 'bearish', 1.30, 'analysis'),
('bull trap', 'bearish', 1.30, 'analysis'),
('ATL', 'bearish', 1.10, 'analysis'),
('all time low', 'bearish', 1.10, 'analysis'),
('new lows', 'bearish', 1.10, 'analysis'),
('ceiling', 'bearish', 0.90, 'analysis'),
('lower lows', 'bearish', 1.00, 'analysis'),
('lower highs', 'bearish', 1.00, 'analysis'),
('gap down', 'bearish', 1.10, 'analysis'),
('death cross', 'bearish', 1.30, 'analysis'),
('head and shoulders', 'bearish', 1.10, 'analysis'),

-- Direct sentiment
('bearish', 'bearish', 1.00, 'direct'),
('bear', 'bearish', 0.90, 'direct'),
('bears', 'bearish', 0.90, 'direct'),

-- Emoji
('📉', 'bearish', 1.30, 'emoji'),
('💩', 'bearish', 1.10, 'emoji'),
('🗑️', 'bearish', 1.20, 'emoji'),
('☠️', 'bearish', 1.30, 'emoji'),
('😭', 'bearish', 0.90, 'emoji'),
('🤡', 'bearish', 1.00, 'emoji'),
('🐻', 'bearish', 1.00, 'emoji'),
('⚠️', 'bearish', 0.80, 'emoji'),
('🔴', 'bearish', 0.90, 'emoji'),
('⬇️', 'bearish', 0.80, 'emoji'),
('💀', 'bearish', 1.20, 'emoji'),
('🪦', 'bearish', 1.30, 'emoji'),
('❌', 'bearish', 0.90, 'emoji'),
('🆘', 'bearish', 1.10, 'emoji'),
('😱', 'bearish', 1.00, 'emoji'),
('🥴', 'bearish', 0.80, 'emoji')

ON CONFLICT (term) DO NOTHING;

-- Modifiers (affect nearby sentiment)
INSERT INTO sentiment_lexicon (term, sentiment, weight, category) VALUES
-- Negation (flip sentiment)
('not', 'modifier', -1.00, 'negation'),
('dont', 'modifier', -1.00, 'negation'),
('do not', 'modifier', -1.00, 'negation'),
('never', 'modifier', -1.00, 'negation'),
('isnt', 'modifier', -1.00, 'negation'),
('is not', 'modifier', -1.00, 'negation'),
('wasnt', 'modifier', -1.00, 'negation'),
('was not', 'modifier', -1.00, 'negation'),
('wont', 'modifier', -1.00, 'negation'),
('will not', 'modifier', -1.00, 'negation'),
('no', 'modifier', -0.80, 'negation'),
('cant', 'modifier', -1.00, 'negation'),
('cannot', 'modifier', -1.00, 'negation'),

-- Amplifiers (strengthen sentiment)
('very', 'modifier', 1.30, 'amplifier'),
('really', 'modifier', 1.20, 'amplifier'),
('extremely', 'modifier', 1.50, 'amplifier'),
('super', 'modifier', 1.30, 'amplifier'),
('absolutely', 'modifier', 1.40, 'amplifier'),
('definitely', 'modifier', 1.30, 'amplifier'),
('clearly', 'modifier', 1.20, 'amplifier'),
('obviously', 'modifier', 1.20, 'amplifier'),
('insanely', 'modifier', 1.40, 'amplifier'),
('massively', 'modifier', 1.40, 'amplifier'),
('hugely', 'modifier', 1.30, 'amplifier'),

-- Reducers (weaken sentiment)
('maybe', 'modifier', 0.50, 'reducer'),
('might', 'modifier', 0.60, 'reducer'),
('could', 'modifier', 0.60, 'reducer'),
('possibly', 'modifier', 0.50, 'reducer'),
('probably', 'modifier', 0.80, 'reducer'),
('perhaps', 'modifier', 0.50, 'reducer'),
('somewhat', 'modifier', 0.70, 'reducer'),
('slightly', 'modifier', 0.60, 'reducer'),
('kind of', 'modifier', 0.60, 'reducer'),
('kinda', 'modifier', 0.60, 'reducer'),
('sorta', 'modifier', 0.60, 'reducer'),
('sort of', 'modifier', 0.60, 'reducer')

ON CONFLICT (term) DO NOTHING;

-- Print summary
DO $$
DECLARE
    bullish_count INTEGER;
    bearish_count INTEGER;
    modifier_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO bullish_count FROM sentiment_lexicon WHERE sentiment = 'bullish';
    SELECT COUNT(*) INTO bearish_count FROM sentiment_lexicon WHERE sentiment = 'bearish';
    SELECT COUNT(*) INTO modifier_count FROM sentiment_lexicon WHERE sentiment = 'modifier';

    RAISE NOTICE 'Sentiment lexicon seeded: % bullish, % bearish, % modifiers',
        bullish_count, bearish_count, modifier_count;
END $$;
