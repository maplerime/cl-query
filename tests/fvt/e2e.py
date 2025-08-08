###
# Licensed Materials - Property of PEG TECH INC
#
# (C) Copyright PEG TECH INC. 2019 ~ 2025 All Rights Reserved
#
# Contributors:
#    bryan@raksmart.com - Initial implementation
###


import requests
import json
import pytest

# Define the base URL
BASE_URL = "http://localhost:8765/query/v1"

API_KEY = "cJGZ8L1sDcPezjOy1zacPJZxzZxrPObm2Ggs1U0V+fE=INSECURE"  # Replace with your actual API key
headers = {"X-API-Key": API_KEY, "Content-Type": "application/json"}


# get version and health
def get_version():
    url = f"{BASE_URL}"
    response = requests.get(url)
    
    assert response.status_code == 200, f"Failed to get query version: {response.status_code}"
    return response.json()

# Tests using pytest
def test_get_version():
    version = get_version()
    assert isinstance(version, dict), "version response is not a dictionary"
    assert 'Version' in version, "version data not found in response"
