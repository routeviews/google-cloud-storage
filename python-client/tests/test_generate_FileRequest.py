import os
import tempfile

from assertpy import assert_that

from routeviews_google_upload import client


def test_remove_leading_backslash():
    # Arrange
    with tempfile.NamedTemporaryFile() as file:
        assert_that(file.name).starts_with(os.sep)

        # Act
        payload = client.generate_FileRequest(file.name, False)

        # Assert
        first_non_os_sep_char = file.name[1]
        assert_that(payload.filename).starts_with(first_non_os_sep_char)
