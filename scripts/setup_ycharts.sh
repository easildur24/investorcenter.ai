#!/bin/bash

# YCharts API Setup Script
# This script sets up the YCharts API query environment

echo "🚀 Setting up YCharts API environment..."
echo

# Check if Python 3 is installed
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is not installed. Please install Python 3 first."
    exit 1
fi

echo "✅ Python 3 found: $(python3 --version)"

# Check if pip3 is available
if ! command -v pip3 &> /dev/null; then
    echo "❌ pip3 is not available. Installing pip..."
    
    # Try to install pip using ensurepip
    python3 -m ensurepip --upgrade
    
    if [ $? -ne 0 ]; then
        echo "❌ Failed to install pip. Please install pip manually:"
        echo "   curl https://bootstrap.pypa.io/get-pip.py -o get-pip.py"
        echo "   python3 get-pip.py"
        exit 1
    fi
fi

echo "✅ pip3 found: $(pip3 --version)"
echo

# Install required packages
echo "📦 Installing required Python packages..."
pip3 install requests pandas python-dotenv

if [ $? -eq 0 ]; then
    echo "✅ All packages installed successfully!"
else
    echo "❌ Failed to install some packages. Please check the error messages above."
    exit 1
fi

echo

# Create .env file if it doesn't exist
if [ ! -f ".env" ]; then
    echo "📝 Creating .env file..."
    cp ycharts_env_example .env
    echo "✅ Created .env file. Please edit it and add your YCharts API key."
else
    echo "ℹ️  .env file already exists."
fi

echo
echo "🎉 Setup complete! Next steps:"
echo "1. Edit the .env file and add your YCharts API key:"
echo "   YCHARTS_API_KEY=your_actual_api_key_here"
echo
echo "2. Test the installation:"
echo "   python3 ycharts_direct_api.py --symbols AAPL --metrics price"
echo "   Or try demo mode: python3 ycharts_direct_api.py --demo --symbols AAPL --metrics price"
echo
echo "3. Run examples:"
echo "   python3 ycharts_examples.py"
echo
echo "4. Read the documentation:"
echo "   cat README_ycharts.md"
