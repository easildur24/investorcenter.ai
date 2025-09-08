// Test script for volume API endpoints
const API_BASE = 'http://localhost:8080/api/v1';

async function testVolumeAPI() {
  console.log('🧪 Testing Volume API Endpoints\n');
  console.log('=' .repeat(50));
  
  // Test 1: Get volume data from database
  console.log('\n1. Testing database volume data (AAPL):');
  try {
    const response = await fetch(`${API_BASE}/tickers/AAPL/volume`);
    const data = await response.json();
    console.log('Response:', JSON.stringify(data, null, 2));
    console.log(`✅ Source: ${data.source}, Real-time: ${data.realtime}`);
  } catch (error) {
    console.log('❌ Error:', error.message);
  }
  
  // Test 2: Get real-time volume data
  console.log('\n2. Testing real-time volume data (AAPL):');
  try {
    const response = await fetch(`${API_BASE}/tickers/AAPL/volume?realtime=true`);
    const data = await response.json();
    console.log('Response:', JSON.stringify(data, null, 2));
    console.log(`✅ Source: ${data.source}, Real-time: ${data.realtime}`);
  } catch (error) {
    console.log('❌ Error:', error.message);
  }
  
  // Test 3: Get volume aggregates
  console.log('\n3. Testing volume aggregates (AAPL):');
  try {
    const response = await fetch(`${API_BASE}/tickers/AAPL/volume/aggregates?days=30`);
    const data = await response.json();
    console.log('Response:', JSON.stringify(data, null, 2));
    console.log(`✅ Volume trend: ${data.data?.volumeTrend}`);
  } catch (error) {
    console.log('❌ Error:', error.message);
  }
  
  // Test 4: Get bulk volume data
  console.log('\n4. Testing bulk volume data:');
  try {
    const response = await fetch(`${API_BASE}/volume/bulk`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ symbols: ['AAPL', 'GOOGL', 'MSFT'] })
    });
    const data = await response.json();
    console.log('Response:', JSON.stringify(data, null, 2));
    console.log(`✅ Retrieved ${data.count} symbols from ${data.source}`);
  } catch (error) {
    console.log('❌ Error:', error.message);
  }
  
  // Test 5: Get top volume stocks
  console.log('\n5. Testing top volume stocks:');
  try {
    const response = await fetch(`${API_BASE}/volume/top?limit=5&type=stock`);
    const data = await response.json();
    console.log('Response:', JSON.stringify(data, null, 2));
    console.log(`✅ Top ${data.count} stocks by volume`);
  } catch (error) {
    console.log('❌ Error:', error.message);
  }
  
  console.log('\n' + '=' .repeat(50));
  console.log('✨ Volume API Test Complete!');
}

// Run the test
testVolumeAPI().catch(console.error);